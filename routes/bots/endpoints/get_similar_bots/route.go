// Package get_similar_bots implements GET /bots/{id}/similar — "Get Similar
// Bots".
//
// Returns other approved/certified bots sharing at least one tag with this
// bot, ranked by how many tags they share
package get_similar_bots

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Similar Bots",
		Description: "Returns other approved/certified bots sharing at least one tag with this bot, ranked by how many tags they share",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: []types.IndexBot{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "id")

	if botId == "" {
		return resp.BadRequest("id is required")
	}

	rows, err := db.New(state.Pool).GetSimilarBots(d.Context, botId)

	if err != nil {
		return resp.Err("Error while getting similar bots [db fetch]", err)
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

	if err := assets.ResolveIndexBots(d.Context, bots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	return uapi.HttpResponse{
		Json: bots,
	}
}
