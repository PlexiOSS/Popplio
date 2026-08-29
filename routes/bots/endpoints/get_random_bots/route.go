// Package get_random_bots implements GET /bots/@random — "Get Random Bots".
//
// Returns a list of bots from the database in random order
package get_random_bots

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Bots",
		Description: "Returns a list of bots from the database in random order",
		Resp: types.RandomBots{
			Bots: []types.IndexBot{},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetRandomIndexBots(d.Context)

	if err != nil {
		return resp.Err("Error while getting random bots [db fetch]", err)
	}

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

	// Resolve all bots concurrently, since each bot's resolution is independent
	if err := assets.ResolveIndexBots(d.Context, bots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	return uapi.HttpResponse{
		Json: types.RandomBots{
			Bots: bots,
		},
	}
}
