package bgtasks

import (
	"context"
	"fmt"

	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

/**
* Bot Uptime Check
*
* This task checks the presence of all listed bots in the main guild and updates their uptime stats
 */
func BotUptimeCheck(ctx context.Context) error {
	rows, err := state.Pool.Query(ctx, "SELECT bot_id FROM bots WHERE type = 'approved' OR type = 'certified'")

	if err != nil {
		return fmt.Errorf("querying listed bots: %w", err)
	}

	var botIDs []string

	for rows.Next() {
		var botID string

		if err := rows.Scan(&botID); err != nil {
			rows.Close()
			return fmt.Errorf("scanning bot_id: %w", err)
		}

		botIDs = append(botIDs, botID)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating listed bots: %w", err)
	}

	mainGuild := state.Config.Servers.Main

	for _, botID := range botIDs {
		userID, err := snowflake.Parse(botID)

		if err != nil {
			state.Logger.Warn("bot_uptime_check: invalid bot_id, skipping", zap.String("botID", botID))
			continue
		}

		online := isOnline(mainGuild, userID)

		_, err = state.Pool.Exec(ctx,
			`UPDATE bots SET
				total_uptime = total_uptime + 1,
				uptime = uptime + CASE WHEN $2 THEN 1 ELSE 0 END,
				uptime_last_checked = NOW()
			WHERE bot_id = $1`,
			botID, online,
		)

		if err != nil {
			return fmt.Errorf("botID=%s: updating uptime: %w", botID, err)
		}
	}

	return nil
}

func isOnline(guildID, userID snowflake.ID) bool {
	presence, ok := state.Discord.Caches().Presence(guildID, userID)

	if !ok {
		return false
	}

	return presence.Status != discord.OnlineStatusOffline && presence.Status != discord.OnlineStatusInvisible
}
