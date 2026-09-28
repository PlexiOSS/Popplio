// Copyright (C) 2026 NodeByte LTD

package botpresence

import (
	"context"
	"errors"
	"time"

	"popplio/state"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix  = "botpresence:"
	unknown    = "unknown"
	FreshTTL   = 15 * time.Minute
	UnknownTTL = time.Hour
)

func Set(ctx context.Context, botID, status string) error {
	return state.Redis.Set(ctx, keyPrefix+botID, status, FreshTTL).Err()
}

func SetUnknown(ctx context.Context, botID string) error {
	return state.Redis.Set(ctx, keyPrefix+botID, unknown, UnknownTTL).Err()
}

func Get(ctx context.Context, botID string) string {
	status, err := state.Redis.Get(ctx, keyPrefix+botID).Result()

	if err != nil || status == unknown {
		return ""
	}

	return status
}

func Tracked(ctx context.Context, botIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(botIDs))

	if len(botIDs) == 0 {
		return out, nil
	}

	keys := make([]string, len(botIDs))
	for i, id := range botIDs {
		keys[i] = keyPrefix + id
	}

	values, err := state.Redis.MGet(ctx, keys...).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	for i, v := range values {
		out[botIDs[i]] = v != nil
	}

	return out, nil
}
