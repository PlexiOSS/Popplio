// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"

	"popplio/db"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

func BotUptimeCheck(ctx context.Context) error {
	q := db.New(state.Pool)

	botIDs, err := q.GetListedBotIDs(ctx)

	if err != nil {
		return fmt.Errorf("querying listed bots: %w", err)
	}

	mainGuild := state.Config.Servers.Main

	for _, botID := range botIDs {
		userID, err := snowflake.Parse(botID)

		if err != nil {
			state.Logger.Warn("bot_uptime_check: invalid bot_id, skipping", zap.String("botID", botID))
			continue
		}

		online := isOnline(mainGuild, userID)

		err = q.RecordBotUptimeCheck(ctx, db.RecordBotUptimeCheckParams{
			Online: online,
			BotID:  botID,
		})

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
