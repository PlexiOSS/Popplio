package assets

import (
	"context"
	"fmt"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/dovewing/dovetypes"
)

const statsPostFreshness = 24 * time.Hour

func ApplySelfStatus(user *dovetypes.PlatformUser, selfStatus string, servers int, lastStatsPost pgtype.Timestamptz) {
	if user == nil {
		return
	}

	if selfStatus != "" {
		user.Status = dovetypes.PlatformStatus(selfStatus)
		return
	}

	if servers > 0 && lastStatsPost.Valid && time.Since(lastStatsPost.Time) < statsPostFreshness {
		user.Status = dovetypes.PlatformStatusOnline
	}
}

func ResolveIndexBot(ctx context.Context, bot *types.IndexBot) error {
	botUser, err := dovewing.GetUser(ctx, bot.BotID, state.DovewingPlatformDiscord)

	if err != nil {
		return fmt.Errorf("error querying for bot user [dovewing]: %w", err)
	}

	bot.User = botUser
	ApplySelfStatus(bot.User, bot.SelfStatus.String, bot.Servers, bot.LastStatsPost)

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

	return g.Wait()
}
