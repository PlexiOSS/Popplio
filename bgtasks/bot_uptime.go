// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"
	"time"

	"popplio/botpresence"
	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

const uptimeStatsFreshness = 24 * time.Hour

func BotUptimeCheck(ctx context.Context) error {
	q := db.New(state.Pool)

	bots, err := q.GetListedBotPresenceInputs(ctx)

	if err != nil {
		return fmt.Errorf("querying listed bots: %w", err)
	}

	mainGuild := state.Config.Servers.Main
	var skipped int

	for _, bot := range bots {
		userID, err := snowflake.Parse(bot.BotID)

		if err != nil {
			state.Logger.Warn("bot_uptime_check: invalid bot_id, skipping", zap.String("botID", bot.BotID))
			continue
		}

		online, known := resolveUptimeStatus(ctx, mainGuild, userID, bot)

		if !known {
			skipped++
			continue
		}

		err = q.RecordBotUptimeCheck(ctx, db.RecordBotUptimeCheckParams{
			Online: online,
			BotID:  bot.BotID,
		})

		if err != nil {
			return fmt.Errorf("botID=%s: updating uptime: %w", bot.BotID, err)
		}
	}

	if skipped > 0 {
		state.Logger.Info("bot_uptime_check: skipped bots with no presence signal", zap.Int("skipped", skipped), zap.Int("total", len(bots)))
	}

	return nil
}

func statusIsOnline(status string) bool {
	switch status {
	case "online", "idle", "dnd":
		return true
	}
	return false
}

func resolveUptimeStatus(ctx context.Context, mainGuild, userID snowflake.ID, bot db.GetListedBotPresenceInputsRow) (online bool, known bool) {
	statsFresh := bot.LastStatsPost.Valid && time.Since(bot.LastStatsPost.Time) < uptimeStatsFreshness

	if bot.SelfStatus.Valid && bot.SelfStatus.String != "" && statsFresh {
		return statusIsOnline(bot.SelfStatus.String), true
	}

	if presence, ok := state.Discord.Caches().Presence(mainGuild, userID); ok {
		return presence.Status != discord.OnlineStatusOffline && presence.Status != discord.OnlineStatusInvisible, true
	}

	if status := botpresence.Get(ctx, bot.BotID); status != "" {
		return statusIsOnline(status), true
	}

	if bot.Servers > 0 && statsFresh {
		return true, true
	}

	return false, false
}
