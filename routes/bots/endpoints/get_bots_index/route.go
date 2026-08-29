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

	botAssets "popplio/routes/bots/assets"
	"popplio/db"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	listIndex := types.ListIndexBot{}

	q := db.New(state.Pool)

	// Certified Bots
	certRows, err := q.GetCertifiedIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting certified bots", err)
	}
	listIndex.Certified, err = processRow(d.Context, toIndexBotsFromCertified(certRows))
	if err != nil {
		return resp.Err("Error while processing certified bots", err)
	}

	// Premium Bots
	premRows, err := q.GetPremiumIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting premium bots", err)
	}
	listIndex.Premium, err = processRow(d.Context, toIndexBotsFromPremium(premRows))
	if err != nil {
		return resp.Err("Error while processing premium bots", err)
	}

	// Most Viewed Bots
	mostViewedRows, err := q.GetMostViewedIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting most viewed bots", err)
	}
	listIndex.MostViewed, err = processRow(d.Context, toIndexBotsFromMostViewed(mostViewedRows))
	if err != nil {
		return resp.Err("Error while processing most viewed bots", err)
	}

	// Recently Added Bots
	recentlyAddedRows, err := q.GetRecentlyAddedIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting recently added bots", err)
	}
	listIndex.RecentlyAdded, err = processRow(d.Context, toIndexBotsFromRecentlyAdded(recentlyAddedRows))
	if err != nil {
		return resp.Err("Error while processing recently added bots", err)
	}

	// Top Voted Bots
	topVotedRows, err := q.GetTopVotedIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting top voted bots", err)
	}
	listIndex.TopVoted, err = processRow(d.Context, toIndexBotsFromTopVoted(topVotedRows))
	if err != nil {
		return resp.Err("Error while processing top voted bots", err)
	}

	// Featured Bots (purchased via the shop, not necessarily approved/certified-ranked)
	featuredRows, err := q.GetFeaturedIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting featured bots", err)
	}
	listIndex.Featured, err = processRow(d.Context, toIndexBotsFromFeatured(featuredRows))
	if err != nil {
		return resp.Err("Error while processing featured bots", err)
	}

	// Spotlight Bots (set by staff, distinct from shop-purchased Featured)
	spotlightRows, err := q.GetSpotlightIndexBots(d.Context)
	if err != nil {
		return resp.Err("Error while getting spotlight bots", err)
	}
	listIndex.Spotlight, err = processRow(d.Context, toIndexBotsFromSpotlight(spotlightRows))
	if err != nil {
		return resp.Err("Error while processing spotlight bots", err)
	}

	// Packs
	packRows, err := q.GetRecentPacks(d.Context)

	if err != nil {
		return resp.Err("Error while getting packs [db fetch]", err)
	}

	listIndex.Packs = make([]types.BotPack, len(packRows))
	for i, row := range packRows {
		listIndex.Packs[i] = types.BotPack{
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

	for i := range listIndex.Packs {
		err = assets.ResolveBotPack(d.Context, &listIndex.Packs[i])

		if err != nil {
			return resp.ErrBody("Error while resolving user pack", "Error resolving user pack.", err, zap.String("url", listIndex.Packs[i].URL))
		}
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
