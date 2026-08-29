// Package assets resolves a bot pack into the bots it contains.
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

	// Ensure these always marshal as `[]` rather than `null` when the pack
	// has no bots/servers/emojis — a nil Go slice serializes to JSON null,
	// which crashes frontend consumers that call .length/.map on it without
	// a null check.
	pack.ResolvedBots = []types.IndexBot{}
	pack.ResolvedServers = []types.IndexServer{}
	pack.Emojis = []types.PackEmoji{}

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

		// Resolve the bot
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

		// Resolve the server
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
			emojis[i] = types.PackEmoji{
				ID:       row.ID,
				Name:     row.Name,
				Animated: row.Animated,
				Position: int(row.Position),
			}
		}

		pack.Emojis = emojis
	}

	pack.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, pack.URL, "pack")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
