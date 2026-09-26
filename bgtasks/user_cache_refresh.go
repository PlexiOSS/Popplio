// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/PlexiOSS/Keel/dovewing"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	userCacheRefreshBatchSize   = 60
	userCacheRefreshWorkers     = 4
	userCacheRefreshLead        = time.Hour
	userCacheRefreshFailBackoff = time.Hour
	userCacheRefreshTimeout     = 20 * time.Second
)

var userCacheRefreshFailures sync.Map

func UserCacheRefresh(ctx context.Context) error {
	now := time.Now()

	userCacheRefreshFailures.Range(func(key, value any) bool {
		if now.Sub(value.(time.Time)) >= userCacheRefreshFailBackoff {
			userCacheRefreshFailures.Delete(key)
		}
		return true
	})

	minAge := max(state.BaseDovewingState.UserExpiryTime-userCacheRefreshLead, 0)

	candidates, err := db.New(state.Pool).GetDiscordUsersDueForRefresh(ctx, db.GetDiscordUsersDueForRefreshParams{
		MinAgeSeconds: minAge.Seconds(),
		BatchSize:     userCacheRefreshBatchSize * 3,
	})

	if err != nil {
		return fmt.Errorf("querying users due for refresh: %w", err)
	}

	due := make([]string, 0, userCacheRefreshBatchSize)

	for _, id := range candidates {
		if _, backingOff := userCacheRefreshFailures.Load(id); backingOff {
			continue
		}

		due = append(due, id)

		if len(due) == userCacheRefreshBatchSize {
			break
		}
	}

	if len(due) == 0 {
		return nil
	}

	var refreshed, failed atomic.Int64

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(userCacheRefreshWorkers)

	for _, id := range due {
		g.Go(func() error {
			rctx, cancel := context.WithTimeout(gctx, userCacheRefreshTimeout)
			defer cancel()

			if _, err := dovewing.RefreshUser(rctx, id, state.DovewingPlatformDiscord); err != nil {
				userCacheRefreshFailures.Store(id, time.Now())
				failed.Add(1)
				state.Logger.Warn("user_cache_refresh: failed to refresh user", zap.String("id", id), zap.Error(err))
				return nil
			}

			userCacheRefreshFailures.Delete(id)
			refreshed.Add(1)
			return nil
		})
	}

	_ = g.Wait()

	state.Logger.Info("user_cache_refresh: batch done",
		zap.Int("due", len(due)),
		zap.Int64("refreshed", refreshed.Load()),
		zap.Int64("failed", failed.Load()),
		zap.Bool("backlog", len(candidates) > len(due)),
	)

	if failed.Load() == int64(len(due)) {
		return fmt.Errorf("all %d user refreshes failed", len(due))
	}

	return nil
}
