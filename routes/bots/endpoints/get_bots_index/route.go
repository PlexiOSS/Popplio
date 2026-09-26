// Package get_bots_index implements GET /bots/@index — "Get Bots Index".
//
// Gets the index of the bot-side of the list. Returns a `ListIndexBot`
// object
package get_bots_index

import (
	"context"
	"fmt"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	botAssets "popplio/routes/bots/assets"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bots Index",
		Description: "Gets the index of the bot-side of the list. Returns a ``ListIndexBot`` object",
		Resp:        types.ListIndexBot{},
	}
}

const packResolveConcurrency = 4

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listIndex := types.ListIndexBot{}

	q := db.New(state.Pool)

	g, ctx := errgroup.WithContext(d.Context)

	section := func(name string, dst *[]types.IndexBot, load func(context.Context) ([]types.IndexBot, error)) {
		g.Go(func() error {
			bots, err := load(ctx)
			if err != nil {
				return fmt.Errorf("getting %s bots: %w", name, err)
			}

			*dst, err = processRow(ctx, bots)
			if err != nil {
				return fmt.Errorf("processing %s bots: %w", name, err)
			}

			return nil
		})
	}

	section("certified", &listIndex.Certified, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetCertifiedIndexBots(ctx)
		return toIndexBotsFromCertified(rows), err
	})
	section("premium", &listIndex.Premium, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetPremiumIndexBots(ctx)
		return toIndexBotsFromPremium(rows), err
	})
	section("most viewed", &listIndex.MostViewed, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetMostViewedIndexBots(ctx)
		return toIndexBotsFromMostViewed(rows), err
	})
	section("recently added", &listIndex.RecentlyAdded, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetRecentlyAddedIndexBots(ctx)
		return toIndexBotsFromRecentlyAdded(rows), err
	})
	section("top voted", &listIndex.TopVoted, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetTopVotedIndexBots(ctx)
		return toIndexBotsFromTopVoted(rows), err
	})
	section("featured", &listIndex.Featured, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetFeaturedIndexBots(ctx)
		return toIndexBotsFromFeatured(rows), err
	})
	section("spotlight", &listIndex.Spotlight, func(ctx context.Context) ([]types.IndexBot, error) {
		rows, err := q.GetSpotlightIndexBots(ctx)
		return toIndexBotsFromSpotlight(rows), err
	})

	g.Go(func() error {
		packRows, err := q.GetRecentPacks(ctx)
		if err != nil {
			return fmt.Errorf("getting packs: %w", err)
		}

		packs := make([]types.BotPack, len(packRows))
		for i, row := range packRows {
			packs[i] = types.BotPack{
				Owner:      row.Owner,
				Name:       row.Name,
				Short:      row.Short,
				Tags:       row.Tags,
				URL:        row.Url,
				CreatedAt:  row.CreatedAt.Time,
				PackType:   row.PackType,
				Bots:       row.Bots,
				Servers:    row.Servers,
				VoteBanned: row.VoteBanned,
			}
		}

		pg, pctx := errgroup.WithContext(ctx)
		pg.SetLimit(packResolveConcurrency)

		for i := range packs {
			pg.Go(func() error {
				if err := assets.ResolveBotPack(pctx, &packs[i]); err != nil {
					return fmt.Errorf("resolving pack %s: %w", packs[i].URL, err)
				}
				return nil
			})
		}

		if err := pg.Wait(); err != nil {
			return err
		}

		listIndex.Packs = packs
		return nil
	})

	if err := g.Wait(); err != nil {
		return resp.Err("Error while building bots index", err, zap.String("route", "get_bots_index"))
	}

	return uapi.HttpResponse{
		Json: listIndex,
	}
}

// processRow validates that every returned bot actually matches the
// approved-or-certified invariant every one of these queries relies on,
// then resolves each bot (user, vanity, votes) concurrently.
func processRow(ctx context.Context, bots []types.IndexBot) ([]types.IndexBot, error) {
	for i := range bots {
		if bots[i].Type != "approved" && bots[i].Type != "certified" {
			return nil, fmt.Errorf("internal error: bot %s has invalid type %s", bots[i].BotID, bots[i].Type)
		}
	}

	// Resolve all bots concurrently, since each bot's resolution is independent
	if err := botAssets.ResolveIndexBots(ctx, bots); err != nil {
		return nil, err
	}

	return bots, nil
}

func toIndexBotsFromCertified(rows []db.GetCertifiedIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromPremium(rows []db.GetPremiumIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromMostViewed(rows []db.GetMostViewedIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromRecentlyAdded(rows []db.GetRecentlyAddedIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromTopVoted(rows []db.GetTopVotedIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromFeatured(rows []db.GetFeaturedIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}

func toIndexBotsFromSpotlight(rows []db.GetSpotlightIndexBotsRow) []types.IndexBot {
	bots := make([]types.IndexBot, len(rows))
	for i, row := range rows {
		bots[i] = types.IndexBot{
			BotID:            row.BotID,
			Short:            row.Short,
			Type:             row.Type,
			VanityRef:        row.VanityRef,
			ApproximateVotes: int(row.ApproximateVotes),
			Shards:           int(row.Shards),
			Library:          row.Library,
			InviteClick:      int(row.InviteClicks),
			Clicks:           int(row.Clicks),
			Servers:          int(row.Servers),
			NSFW:             row.Nsfw,
			Tags:             row.Tags,
			Premium:          row.Premium,
			CreatedAt:        row.CreatedAt,
			SelfStatus:       row.SelfStatus,
			LastStatsPost:    row.LastStatsPost,
			SupporterBadge:   row.SupporterBadge,
			BoostedUntil:     row.BoostedUntil,
			FeaturedUntil:    row.FeaturedUntil,
			SpotlightedUntil: row.SpotlightedUntil,
			VoteBlitzUntil:   row.VoteBlitzUntil,
		}
	}
	return bots
}
