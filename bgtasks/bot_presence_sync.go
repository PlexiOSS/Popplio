// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"popplio/botpresence"
	"popplio/db"
	"popplio/japi"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

const (
	botPresenceBatchSize = 200
	botPresenceGap       = 1200 * time.Millisecond
)

func BotPresenceSync(ctx context.Context) error {
	botIDs, err := db.New(state.Pool).GetListedBotIDs(ctx)

	if err != nil {
		return fmt.Errorf("querying listed bots: %w", err)
	}

	mainGuild := state.Config.Servers.Main
	candidates := make([]string, 0, len(botIDs))

	for _, botID := range botIDs {
		userID, err := snowflake.Parse(botID)

		if err != nil {
			continue
		}

		if _, ok := state.Discord.Caches().Presence(mainGuild, userID); ok {
			continue
		}

		candidates = append(candidates, botID)
	}

	tracked, err := botpresence.Tracked(ctx, candidates)

	if err != nil {
		return fmt.Errorf("reading tracked presences: %w", err)
	}

	due := make([]string, 0, botPresenceBatchSize)
	untracked := 0

	for _, botID := range candidates {
		if tracked[botID] {
			continue
		}

		untracked++

		if len(due) < botPresenceBatchSize {
			due = append(due, botID)
		}
	}

	if len(due) == 0 {
		return nil
	}

	var updated, unknown, failed int64
	rateLimited := false

	pace := time.NewTicker(botPresenceGap)
	defer pace.Stop()

	for i, botID := range due {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-pace.C:
			}
		}

		presence, err := japi.GetPresence(ctx, botID)

		switch {
		case err == nil:
			if err := botpresence.Set(ctx, botID, presence.Status); err != nil {
				failed++
				continue
			}
			updated++
		case errors.Is(err, japi.ErrNotFound):
			if err := botpresence.SetUnknown(ctx, botID); err != nil {
				failed++
				continue
			}
			unknown++
		case errors.Is(err, japi.ErrUnavailable), errors.Is(err, japi.ErrRateLimited):
			rateLimited = true
			state.Logger.Info("bot_presence_sync: japi.rest asked us to slow down, stopping this batch", zap.Error(err))
		default:
			failed++
			state.Logger.Warn("bot_presence_sync: presence lookup failed", zap.String("botID", botID), zap.Error(err))
		}

		if rateLimited {
			break
		}
	}

	state.Logger.Info("bot_presence_sync: batch done",
		zap.Int("due", len(due)),
		zap.Int64("updated", updated),
		zap.Int64("unknown", unknown),
		zap.Int64("failed", failed),
		zap.Bool("backlog", untracked > len(due)),
		zap.Bool("rate_limited", rateLimited),
	)

	if updated == 0 && unknown == 0 && failed > 0 {
		return fmt.Errorf("no bot presences could be fetched from japi.rest (%d failed)", failed)
	}

	return nil
}
