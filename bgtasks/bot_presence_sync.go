// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"popplio/botpresence"
	"popplio/db"
	"popplio/japi"
	"popplio/state"

	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	botPresenceBatchSize = 100
	botPresenceWorkers   = 4
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

	var updated, unknown, failed atomic.Int64

	runCtx, stop := context.WithCancel(ctx)
	defer stop()

	g, gctx := errgroup.WithContext(runCtx)
	g.SetLimit(botPresenceWorkers)

	for _, botID := range due {
		g.Go(func() error {
			if gctx.Err() != nil {
				return nil
			}

			presence, err := japi.GetPresence(gctx, botID)

			switch {
			case err == nil:
				if err := botpresence.Set(ctx, botID, presence.Status); err != nil {
					failed.Add(1)
					return nil
				}
				updated.Add(1)
			case errors.Is(err, japi.ErrNotFound):
				if err := botpresence.SetUnknown(ctx, botID); err != nil {
					failed.Add(1)
					return nil
				}
				unknown.Add(1)
			case errors.Is(err, japi.ErrUnavailable), errors.Is(err, japi.ErrRateLimited):
				failed.Add(1)
				stop()
			default:
				failed.Add(1)
				state.Logger.Warn("bot_presence_sync: presence lookup failed", zap.String("botID", botID), zap.Error(err))
			}

			return nil
		})
	}

	_ = g.Wait()

	state.Logger.Info("bot_presence_sync: batch done",
		zap.Int("due", len(due)),
		zap.Int64("updated", updated.Load()),
		zap.Int64("unknown", unknown.Load()),
		zap.Int64("failed", failed.Load()),
		zap.Bool("backlog", untracked > len(due)),
	)

	if updated.Load() == 0 && unknown.Load() == 0 && failed.Load() > 0 {
		return fmt.Errorf("no bot presences could be fetched from japi.rest (%d failed)", failed.Load())
	}

	return nil
}
