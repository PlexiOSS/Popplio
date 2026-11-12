// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	arcadiadclient "popplio/arcadia/dclient"
	"popplio/botpresence"
	"popplio/db"
	"popplio/japi"
	"popplio/state"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"go.uber.org/zap"
)

const (
	botPresenceBatchSize = 200
	botPresenceGap       = 1200 * time.Millisecond
)

func pickDueBots(candidates []string, tracked map[string]bool, limit int) ([]string, int) {
	due := make([]string, 0, len(candidates))

	for _, botID := range candidates {
		if !tracked[botID] {
			due = append(due, botID)
		}
	}

	untracked := len(due)

	rand.Shuffle(untracked, func(i, j int) {
		due[i], due[j] = due[j], due[i]
	})

	return due[:min(limit, untracked)], untracked
}

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

		if _, ok := state.Discord.Caches.Presence(mainGuild, userID); ok {
			continue
		}

		candidates = append(candidates, botID)
	}

	candidates = applyArcadiaPresences(ctx, candidates)

	tracked, err := botpresence.Tracked(ctx, candidates)

	if err != nil {
		return fmt.Errorf("reading tracked presences: %w", err)
	}

	due, untracked := pickDueBots(candidates, tracked, botPresenceBatchSize)

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

func applyArcadiaPresences(ctx context.Context, candidates []string) []string {
	seen := arcadiaPresences(candidates)

	if len(seen) == 0 {
		return candidates
	}

	rest := candidates[:0:0]
	stored := 0

	for _, botID := range candidates {
		status, ok := seen[botID]

		if !ok {
			rest = append(rest, botID)
			continue
		}

		if err := botpresence.Set(ctx, botID, status); err != nil {
			rest = append(rest, botID)
			continue
		}

		stored++
	}

	state.Logger.Info("bot_presence_sync: read presences from Arcadia's servers", zap.Int("bots", stored))

	return rest
}

func arcadiaPresences(botIDs []string) map[string]string {
	out := map[string]string{}

	if !arcadiadclient.Ready() {
		return out
	}

	client := arcadiadclient.Get()
	want := make(map[snowflake.ID]string, len(botIDs))

	for _, id := range botIDs {
		if sf, err := snowflake.Parse(id); err == nil {
			want[sf] = id
		}
	}

	type sighting struct {
		guild, user snowflake.ID
	}

	var guilds []snowflake.ID

	for g := range client.Caches.Guilds() {
		guilds = append(guilds, g.ID)
	}

	var found []sighting

	for _, guildID := range guilds {
		for m := range client.Caches.Members(guildID) {
			if _, ok := want[m.User.ID]; ok {
				found = append(found, sighting{guild: guildID, user: m.User.ID})
			}
		}
	}

	for _, f := range found {
		status := presenceStatus(client.Caches.Presence(f.guild, f.user))
		id := want[f.user]

		if prev, ok := out[id]; !ok || prev == "offline" {
			out[id] = status
		}
	}

	return out
}

func presenceStatus(p discord.Presence, ok bool) string {
	if !ok || p.Status == discord.OnlineStatusOffline || p.Status == discord.OnlineStatusInvisible {
		return "offline"
	}

	return string(p.Status)
}
