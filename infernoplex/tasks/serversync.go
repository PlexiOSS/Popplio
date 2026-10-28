// Copyright (C) 2026 NodeByte LTD

package tasks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func ServerSync(ctx context.Context) error {
	metaErr := syncServerMeta(ctx)

	if metaErr != nil {
		state.Logger.Error("Server sync: meta pass failed, continuing with emoji/sticker pass", zap.Error(metaErr))
	}

	return errors.Join(metaErr, syncServerEmojisStickers(ctx))
}

func syncServerEmojisStickers(ctx context.Context) error {
	q := db.New(state.Pool)

	serverIDs, err := q.GetServersWithEmojisShown(ctx)

	if err != nil {
		return err
	}

	var failed int

	for _, serverID := range serverIDs {
		guildID, err := snowflake.Parse(serverID)

		if err != nil {
			continue
		}

		emojis, err := dclient.Get().Rest().GetEmojis(guildID)

		if err != nil {
			state.Logger.Warn("Server sync: failed to fetch emojis, skipping", zap.Error(err), zap.String("server_id", serverID))
			continue
		}

		stickers, err := dclient.Get().Rest().GetStickers(guildID)

		if err != nil {
			state.Logger.Warn("Server sync: failed to fetch stickers, skipping", zap.Error(err), zap.String("server_id", serverID))
			continue
		}

		syncedEmojis := make([]types.Emoji, 0, len(emojis))

		for _, e := range emojis {
			url := e.URL()
			if e.Animated {
				url = e.URL(discord.WithFormat(discord.FileFormatGIF))
			}

			syncedEmojis = append(syncedEmojis, types.Emoji{
				ID:       e.ID.String(),
				Name:     e.Name,
				Animated: e.Animated,
				URL:      url,
			})
		}

		syncedStickers := make([]types.Sticker, 0, len(stickers))

		for _, s := range stickers {
			syncedStickers = append(syncedStickers, types.Sticker{
				ID:     s.ID.String(),
				Name:   s.Name,
				Format: stickerFormatName(s.FormatType),
				URL:    s.URL(),
			})
		}

		if err := q.UpdateServerEmojisStickers(ctx, db.UpdateServerEmojisStickersParams{
			ServerID: serverID,
			Emojis:   syncedEmojis,
			Stickers: syncedStickers,
		}); err != nil {
			state.Logger.Error("Server sync: failed to save emojis/stickers, skipping", zap.Error(err), zap.String("server_id", serverID))
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("emoji/sticker sync failed to save %d of %d servers", failed, len(serverIDs))
	}

	return nil
}

func stickerFormatName(t discord.StickerFormatType) string {
	switch t {
	case discord.StickerFormatTypePNG:
		return "png"
	case discord.StickerFormatTypeAPNG:
		return "apng"
	case discord.StickerFormatTypeLottie:
		return "lottie"
	case discord.StickerFormatTypeGIF:
		return "gif"
	default:
		return "unknown"
	}
}

type syncTarget struct {
	serverID         string
	statsSelfManaged bool
	invite           string
}

const metaSyncWorkers = 4

var inviteCodeRegex = regexp.MustCompile(`(?:discord(?:\.gg|\.com/invite|app\.com/invite)/)?([a-zA-Z0-9-]+)/?$`)

func syncFromInvite(ctx context.Context, q *db.Queries, target syncTarget) error {
	match := inviteCodeRegex.FindStringSubmatch(strings.TrimSpace(target.invite))

	if len(match) < 2 || match[1] == "" {
		return errors.New("no usable invite code")
	}

	inv, err := dclient.Get().Rest().GetInvite(match[1], rest.WithQueryParam("with_counts", true))

	if err != nil {
		return err
	}

	if inv.Guild == nil || inv.Guild.ID.String() != target.serverID {
		return errors.New("invite points at a different server")
	}

	return q.UpdateServerMemberCounts(ctx, db.UpdateServerMemberCountsParams{
		ServerID:      target.serverID,
		TotalMembers:  int32(inv.ApproximateMemberCount),
		OnlineMembers: int32(inv.ApproximatePresenceCount),
	})
}

func syncServerMeta(ctx context.Context) error {
	q := db.New(state.Pool)

	rows, err := q.GetServerIDsAndStatsSelfManaged(ctx)

	if err != nil {
		return err
	}

	targets := make([]syncTarget, len(rows))
	for i, row := range rows {
		targets[i] = syncTarget{serverID: row.ServerID, statsSelfManaged: row.StatsSelfManaged, invite: row.Invite}
	}

	var failed atomic.Int64

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(metaSyncWorkers)

	for _, target := range targets {
		g.Go(func() error {
			if syncServerTarget(gctx, q, target) {
				failed.Add(1)
			}
			return nil
		})
	}

	_ = g.Wait()

	if n := failed.Load(); n > 0 {
		return fmt.Errorf("meta sync failed to save %d of %d servers", n, len(targets))
	}

	return nil
}

func syncServerTarget(ctx context.Context, q *db.Queries, target syncTarget) bool {
	serverID := target.serverID
	guildID, err := snowflake.Parse(serverID)

	if err != nil {
		return false
	}

	guild, err := dclient.Get().Rest().GetGuild(guildID, true)

	if err != nil {
		if target.statsSelfManaged {
			state.Logger.Warn("Server sync: failed to fetch guild, keeping last-known metadata", zap.Error(err), zap.String("server_id", serverID))
			return false
		}

		if invErr := syncFromInvite(ctx, q, target); invErr != nil {
			state.Logger.Warn("Server sync: guild and invite both unavailable, keeping last-known metadata", zap.Error(err), zap.NamedError("invite_error", invErr), zap.String("server_id", serverID))
		}

		return false
	}

	avatar := ""

	if url := guild.IconURL(); url != nil {
		avatar = *url
	}

	var nsfwChannelCount pgtype.Int4

	if channels, err := dclient.Get().Rest().GetGuildChannels(guildID); err == nil {
		nsfwChannelCount = pgtype.Int4{Int32: int32(countNSFWChannels(channels)), Valid: true}
	} else {
		state.Logger.Warn("Server sync: failed to fetch channels, leaving nsfw_channel_count untouched", zap.Error(err), zap.String("server_id", serverID))
	}

	if target.statsSelfManaged {
		if err := q.UpdateServerAvatarAndNsfwStats(ctx, db.UpdateServerAvatarAndNsfwStatsParams{
			ServerID:         serverID,
			Avatar:           avatar,
			PremiumTier:      int16(guild.PremiumTier),
			BoostCount:       int32(guild.PremiumSubscriptionCount),
			DiscordNsfwLevel: int16(guild.NSFWLevel),
			NsfwChannelCount: nsfwChannelCount,
		}); err != nil {
			state.Logger.Error("Server sync: failed to save server metadata, skipping", zap.Error(err), zap.String("server_id", serverID))
			return true
		}
		return false
	}

	if err := q.UpdateServerAvatarMembersAndNsfw(ctx, db.UpdateServerAvatarMembersAndNsfwParams{
		ServerID:         serverID,
		Avatar:           avatar,
		PremiumTier:      int16(guild.PremiumTier),
		BoostCount:       int32(guild.PremiumSubscriptionCount),
		TotalMembers:     int32(guild.ApproximateMemberCount),
		OnlineMembers:    int32(guild.ApproximatePresenceCount),
		DiscordNsfwLevel: int16(guild.NSFWLevel),
		NsfwChannelCount: nsfwChannelCount,
	}); err != nil {
		state.Logger.Error("Server sync: failed to save server metadata, skipping", zap.Error(err), zap.String("server_id", serverID))
		return true
	}

	return false
}

func countNSFWChannels(channels []discord.GuildChannel) int {
	count := 0

	for _, ch := range channels {
		nsfw := false

		switch c := ch.(type) {
		case discord.GuildTextChannel:
			nsfw = c.NSFW()
		case discord.GuildVoiceChannel:
			nsfw = c.NSFW()
		case discord.GuildStageVoiceChannel:
			nsfw = c.NSFW()
		case discord.GuildNewsChannel:
			nsfw = c.NSFW()
		case discord.GuildForumChannel:
			nsfw = c.NSFW
		case discord.GuildMediaChannel:
			nsfw = c.NSFW
		}

		if nsfw {
			count++
		}
	}

	return count
}
