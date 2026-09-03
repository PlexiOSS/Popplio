// Copyright (C) 2026 NodeByte LTD

package tasks

import (
	"context"

	"popplio/db"
	"popplio/infernoplex/dclient"
	"popplio/state"
	"popplio/types"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func ServerSync(ctx context.Context) error {
	if err := syncServerMeta(ctx); err != nil {
		return err
	}

	q := db.New(state.Pool)

	serverIDs, err := q.GetServersWithEmojisShown(ctx)

	if err != nil {
		return err
	}

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
			return err
		}
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
}

func syncServerMeta(ctx context.Context) error {
	q := db.New(state.Pool)

	rows, err := q.GetServerIDsAndStatsSelfManaged(ctx)

	if err != nil {
		return err
	}

	targets := make([]syncTarget, len(rows))
	for i, row := range rows {
		targets[i] = syncTarget{serverID: row.ServerID, statsSelfManaged: row.StatsSelfManaged}
	}

	for _, target := range targets {
		serverID := target.serverID
		guildID, err := snowflake.Parse(serverID)

		if err != nil {
			continue
		}

		guild, err := dclient.Get().Rest().GetGuild(guildID, true)

		if err != nil {
			continue
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
				DiscordNsfwLevel: int16(guild.NSFWLevel),
				NsfwChannelCount: nsfwChannelCount,
			}); err != nil {
				return err
			}
			continue
		}

		if err := q.UpdateServerAvatarMembersAndNsfw(ctx, db.UpdateServerAvatarMembersAndNsfwParams{
			ServerID:         serverID,
			Avatar:           avatar,
			TotalMembers:     int32(guild.ApproximateMemberCount),
			OnlineMembers:    int32(guild.ApproximatePresenceCount),
			DiscordNsfwLevel: int16(guild.NSFWLevel),
			NsfwChannelCount: nsfwChannelCount,
		}); err != nil {
			return err
		}
	}

	return nil
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
