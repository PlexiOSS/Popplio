// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"fmt"

	"popplio/db"
	botassets "popplio/routes/bots/assets"
	serverassets "popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/dovewing"
)

func ResolveBotPack(ctx context.Context, pack *types.BotPack) error {
	ownerUser, err := dovewing.GetUser(ctx, pack.Owner, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying dovewing for owner user: %w", err)
	}

	pack.ResolvedOwner = ownerUser
	pack.ResolvedBots = []types.IndexBot{}
	pack.ResolvedServers = []types.IndexServer{}
	pack.Emojis = []types.PackEmoji{}
	pack.Stickers = []types.PackSticker{}

	q := db.New(state.Pool)

	for _, botId := range pack.Bots {
		row, err := q.GetIndexBotByID(ctx, botId)

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			state.Logger.Error("Error querying bots table [db fetch]", zap.Error(err), zap.String("bot_id", botId))
			return fmt.Errorf("error querying bots table: %w", err)
		}

		bot := types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}

		err = botassets.ResolveIndexBot(ctx, &bot)

		if err != nil {
			return fmt.Errorf("error occurred while resolving index bot %s: %w", bot.BotID, err)
		}

		pack.ResolvedBots = append(pack.ResolvedBots, bot)
	}

	for _, serverId := range pack.Servers {
		row, err := q.GetIndexServerByID(ctx, serverId)

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			state.Logger.Error("Error querying servers table [db fetch]", zap.Error(err), zap.String("server_id", serverId))
			return fmt.Errorf("error querying servers table: %w", err)
		}

		server := types.IndexServer{
			ServerID:         row.ServerID,
			Name:             row.Name,
			Avatar:           row.Avatar,
			TotalMembers:     int(row.TotalMembers),
			OnlineMembers:    int(row.OnlineMembers),
			Short:            row.Short,
			Type:             row.Type,
			State:            row.State,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			InviteClicks:     int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
		}

		err = serverassets.ResolveIndexServer(ctx, &server)

		if err != nil {
			return fmt.Errorf("error occurred while resolving index server %s: %w", server.ServerID, err)
		}

		pack.ResolvedServers = append(pack.ResolvedServers, server)
	}

	if pack.PackType == types.PackTypeEmoji {
		rows, err := q.GetPackEmojis(ctx, pack.URL)

		if err != nil {
			state.Logger.Error("Error querying pack_emojis table [db fetch]", zap.Error(err), zap.String("pack_url", pack.URL))
			return fmt.Errorf("error querying pack_emojis table: %w", err)
		}

		emojis := make([]types.PackEmoji, len(rows))
		for i, row := range rows {
			vanity, err := ResolveVanityCode(ctx, q, row.ID)

			if err != nil {
				return fmt.Errorf("error resolving vanity for pack emoji %s: %w", row.ID, err)
			}

			emojis[i] = types.PackEmoji{
				ID:        row.ID,
				Name:      row.Name,
				Animated:  row.Animated,
				Position:  int(row.Position),
				Downloads: int(row.Downloads),
				Vanity:    vanity,
			}
		}

		pack.Emojis = emojis
	}

	if pack.PackType == types.PackTypeSticker {
		rows, err := q.GetPackStickers(ctx, pack.URL)

		if err != nil {
			state.Logger.Error("Error querying pack_stickers table [db fetch]", zap.Error(err), zap.String("pack_url", pack.URL))
			return fmt.Errorf("error querying pack_stickers table: %w", err)
		}

		stickers := make([]types.PackSticker, len(rows))
		for i, row := range rows {
			vanity, err := ResolveVanityCode(ctx, q, row.ID)

			if err != nil {
				return fmt.Errorf("error resolving vanity for pack sticker %s: %w", row.ID, err)
			}

			stickers[i] = types.PackSticker{
				ID:        row.ID,
				Name:      row.Name,
				Animated:  row.Animated,
				Position:  int(row.Position),
				Downloads: int(row.Downloads),
				Vanity:    vanity,
			}
		}

		pack.Stickers = stickers
	}

	pack.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, pack.URL, "pack")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
