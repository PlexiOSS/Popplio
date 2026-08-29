// Package get_all_bots implements GET /bots/@all — "Get All Bots".
//
// Gets all bots on the list. Returns a set of paginated `IndexBot` objects
package get_all_bots

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Bots",
		Description: "Gets all bots on the list. Returns a set of paginated ``IndexBot`` objects",
		Resp:        types.PagedResult[[]types.IndexBot]{},
		RespName:    "PagedResultIndexBot",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "sort",
				Description: "Sort order. Omit for newest-first; \"trending\" ranks by net votes in the last 7 days instead, and only returns bots with at least one vote in that window.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage
	trending := r.URL.Query().Get("sort") == "trending"

	q := db.New(state.Pool)

	var bots []types.IndexBot

	if trending {
		rows, err := q.GetTrendingIndexBots(d.Context, db.GetTrendingIndexBotsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})

		if err != nil {
			return resp.Err("Error while getting all bots [db fetch]", err)
		}

		bots = make([]types.IndexBot, len(rows))
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
	} else {
		rows, err := q.GetIndexBotsPaged(d.Context, db.GetIndexBotsPagedParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})

		if err != nil {
			return resp.Err("Error while getting all bots [db fetch]", err)
		}

		bots = make([]types.IndexBot, len(rows))
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
	}

	// Resolve all bots concurrently, since each bot's resolution is independent
	if err := assets.ResolveIndexBots(d.Context, bots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	var countRaw int64

	if trending {
		countRaw, err = q.CountTrendingBots(d.Context)
	} else {
		countRaw, err = q.CountBots(d.Context)
	}

	if err != nil {
		return resp.Err("Error while getting bot count", err)
	}

	count := uint64(countRaw)

	data := types.PagedResult[[]types.IndexBot]{
		Count:   count,
		Results: bots,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
