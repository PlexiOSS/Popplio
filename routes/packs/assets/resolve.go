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
	"strings"

	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	indexBotColArr = db.GetCols(types.IndexBot{})
	indexBotCols   = strings.Join(indexBotColArr, ",")

	indexServerColArr = db.GetCols(types.IndexServer{})
	indexServerCols   = strings.Join(indexServerColArr, ",")

	packEmojiColArr = db.GetCols(types.PackEmoji{})
	packEmojiCols   = strings.Join(packEmojiColArr, ",")
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

	for _, botId := range pack.Bots {
		row, err := state.Pool.Query(ctx, "SELECT "+indexBotCols+" FROM bots WHERE bot_id = $1", botId)

		if err != nil {
			state.Logger.Error("Error querying bots table [db fetch]", zap.Error(err), zap.String("bot_id", botId))
			return fmt.Errorf("error querying bots table: %w", err)
		}

		bot, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.IndexBot])

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			return fmt.Errorf("error querying bots table: %w", err)
		}

		// Resolve the bot
		err = botassets.ResolveIndexBot(ctx, &bot)

		if err != nil {
			return fmt.Errorf("error occurred while resolving index bot %s: %w", bot.BotID, err)
		}

		pack.ResolvedBots = append(pack.ResolvedBots, bot)
	}

	for _, serverId := range pack.Servers {
		row, err := state.Pool.Query(ctx, "SELECT "+indexServerCols+" FROM servers WHERE server_id = $1", serverId)

		if err != nil {
			state.Logger.Error("Error querying servers table [db fetch]", zap.Error(err), zap.String("server_id", serverId))
			return fmt.Errorf("error querying servers table: %w", err)
		}

		server, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.IndexServer])

		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}

		if err != nil {
			return fmt.Errorf("error querying servers table: %w", err)
		}

		// Resolve the server
		err = serverassets.ResolveIndexServer(ctx, &server)

		if err != nil {
			return fmt.Errorf("error occurred while resolving index server %s: %w", server.ServerID, err)
		}

		pack.ResolvedServers = append(pack.ResolvedServers, server)
	}

	if pack.PackType == types.PackTypeEmoji {
		rows, err := state.Pool.Query(ctx, "SELECT "+packEmojiCols+" FROM pack_emojis WHERE pack_url = $1 ORDER BY position ASC", pack.URL)

		if err != nil {
			state.Logger.Error("Error querying pack_emojis table [db fetch]", zap.Error(err), zap.String("pack_url", pack.URL))
			return fmt.Errorf("error querying pack_emojis table: %w", err)
		}

		emojis, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.PackEmoji])

		if err != nil {
			return fmt.Errorf("error collecting pack emojis: %w", err)
		}

		pack.Emojis = emojis
	}

	pack.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, pack.URL, "pack")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}
