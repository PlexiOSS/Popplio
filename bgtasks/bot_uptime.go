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
	"github.com/jackc/pgx/v5/pgtype"
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

	if err := q.PruneBotUptimeDaily(ctx); err != nil {
		state.Logger.Warn("bot_uptime_check: failed to prune old daily counts", zap.Error(err))
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

type uptimeSignals struct {
	selfStatus     string
	statsFresh     bool
	servers        int32
	presence       string
	lastSeenOnline pgtype.Timestamptz
}

func decideUptime(s uptimeSignals) (online bool, known bool) {
	if s.selfStatus != "" && s.statsFresh {
		return statusIsOnline(s.selfStatus), true
	}

	if statusIsOnline(s.presence) {
		return true, true
	}

	if s.servers > 0 && s.statsFresh {
		return true, true
	}

	if s.presence == "offline" && botpresence.OfflineIsTrusted(s.lastSeenOnline) {
		return false, true
	}

	return false, false
}

func resolveUptimeStatus(ctx context.Context, mainGuild, userID snowflake.ID, bot db.GetListedBotPresenceInputsRow) (online bool, known bool) {
	signals := uptimeSignals{
		selfStatus:     bot.SelfStatus.String,
		statsFresh:     bot.LastStatsPost.Valid && time.Since(bot.LastStatsPost.Time) < uptimeStatsFreshness,
		servers:        bot.Servers,
		lastSeenOnline: bot.LastSeenOnline,
	}

	if presence, ok := state.Discord.Caches.Presence(mainGuild, userID); ok {
		signals.presence = "online"

		if presence.Status == discord.OnlineStatusOffline || presence.Status == discord.OnlineStatusInvisible {
			signals.presence = "offline"
		}
	} else {
		signals.presence = botpresence.Get(ctx, bot.BotID)
	}

	return decideUptime(signals)
}
