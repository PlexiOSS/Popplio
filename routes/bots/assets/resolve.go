package assets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"popplio/api"
	"popplio/botpresence"
	"popplio/db"
	"popplio/entityassets"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

const statsPostFreshness = 24 * time.Hour

func ApplySelfStatus(user *dovetypes.PlatformUser, selfStatus string, servers int, lastStatsPost pgtype.Timestamptz, japiStatus string) {
	if user == nil {
		return
	}

	if selfStatus != "" {
		user.Status = dovetypes.PlatformStatus(selfStatus)
		return
	}

	if cache, _ := user.ExtraData["cache"].(string); cache == "platform" {
		return
	}

	if status, ok := japiPresenceStatus(japiStatus); ok {
		user.Status = status
		return
	}

	if servers > 0 && lastStatsPost.Valid && time.Since(lastStatsPost.Time) < statsPostFreshness {
		user.Status = dovetypes.PlatformStatusOnline
		return
	}

	user.Status = dovetypes.PlatformStatusOffline
}

func japiPresenceStatus(status string) (dovetypes.PlatformStatus, bool) {
	switch status {
	case "online":
		return dovetypes.PlatformStatusOnline, true
	case "idle":
		return dovetypes.PlatformStatusIdle, true
	case "dnd":
		return dovetypes.PlatformStatusDoNotDisturb, true
	case "offline", "invisible":
		return dovetypes.PlatformStatusOffline, true
	default:
		return "", false
	}
}

func ResolveIndexBot(ctx context.Context, bot *types.IndexBot) error {
	botUser, err := dovewing.GetUser(ctx, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying for bot user [dovewing]: %w", err)
	}

	bot.User = botUser
	ApplySelfStatus(bot.User, bot.SelfStatus.String, bot.Servers, bot.LastStatsPost, botpresence.Get(ctx, bot.BotID))

	code, err := db.New(state.Pool).GetVanityCodeByItag(ctx, bot.VanityRef)

	if err != nil {
		return fmt.Errorf("error querying vanity table: %w", err)
	}

	bot.Vanity = code

	bot.Votes, err = votes.EntityGetVoteCount(ctx, state.Pool, bot.BotID, "bot")

	if err != nil {
		return fmt.Errorf("error getting vote count: %w", err)
	}

	return nil
}

func ResolveIndexBots(ctx context.Context, bots []types.IndexBot) error {
	g, ctx := errgroup.WithContext(ctx)

	for i := range bots {
		g.Go(func() error {
			if err := ResolveIndexBot(ctx, &bots[i]); err != nil {
				return fmt.Errorf("botID=%s: %w", bots[i].BotID, err)
			}
			return nil
		})
	}

	g.Go(func() error {
		ids := make([]string, len(bots))
		for i := range bots {
			ids[i] = bots[i].BotID
		}

		versions, err := entityassets.GetMany(ctx, api.TargetTypeBot, ids)
		if err != nil {
			return fmt.Errorf("error getting asset versions: %w", err)
		}

		for i := range bots {
			bots[i].AssetVersions = versions[bots[i].BotID]
		}

		return nil
	})

	return g.Wait()
}

func ResolveBotChangelogFeedEntries(ctx context.Context, entries []types.BotChangelogFeedEntry) error {
	uniqueBotIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		uniqueBotIDs[entry.BotID] = struct{}{}
	}

	users := make(map[string]*dovetypes.PlatformUser, len(uniqueBotIDs))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	for botID := range uniqueBotIDs {
		g.Go(func() error {
			botUser, err := dovewing.GetUser(ctx, botID, state.DovewingPlatformDiscord)
			if err != nil {
				return fmt.Errorf("botID=%s: %w", botID, err)
			}

			mu.Lock()
			users[botID] = botUser
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for i := range entries {
		entries[i].User = users[entries[i].BotID]
	}

	return nil
}

func ResolveBotCommandSearchResults(ctx context.Context, results []types.BotCommandSearchResult) error {
	uniqueBotIDs := make(map[string]struct{}, len(results))
	for _, result := range results {
		uniqueBotIDs[result.BotID] = struct{}{}
	}

	users := make(map[string]*dovetypes.PlatformUser, len(uniqueBotIDs))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	for botID := range uniqueBotIDs {
		g.Go(func() error {
			botUser, err := dovewing.GetUser(ctx, botID, state.DovewingPlatformDiscord)
			if err != nil {
				return fmt.Errorf("botID=%s: %w", botID, err)
			}

			mu.Lock()
			users[botID] = botUser
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for i := range results {
		results[i].Bot = users[results[i].BotID]
	}

	return nil
}
