// Package get_all_bots implements GET /bots/@all — "Get All Bots".
//
// Gets all bots on the list. Returns a set of paginated `IndexBot` objects
package get_all_bots

import (
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/pagination"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const perPage = 12

var (
	indexBotColsArr = db.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

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

const trendingWindow = "7 days"

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage
	trending := r.URL.Query().Get("sort") == "trending"

	var rows pgx.Rows

	if trending {
		rows, err = state.Pool.Query(d.Context, `
			WITH scored AS (
				SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS score
				FROM entity_votes
				WHERE target_type = 'bot' AND void = false AND created_at > now() - interval '`+trendingWindow+`'
				GROUP BY target_id
			)
			SELECT `+indexBotCols+` FROM bots
			WHERE (type = 'approved' OR type = 'certified') AND bot_id IN (SELECT target_id FROM scored)
			ORDER BY (SELECT score FROM scored WHERE scored.target_id = bots.bot_id) DESC
			LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		rows, err = state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	}

	if err != nil {
		return resp.Err("Error while getting all bots [db fetch]", err)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		return resp.Err("Error while getting all bots [collect]", err)
	}

	// Resolve all bots concurrently, since each bot's resolution is independent
	if err := assets.ResolveIndexBots(d.Context, bots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	var count uint64

	if trending {
		err = state.Pool.QueryRow(d.Context, `
			SELECT COUNT(*) FROM bots
			WHERE (type = 'approved' OR type = 'certified') AND bot_id IN (
				SELECT target_id FROM entity_votes
				WHERE target_type = 'bot' AND void = false AND created_at > now() - interval '`+trendingWindow+`'
				GROUP BY target_id
			)`).Scan(&count)
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM bots").Scan(&count)
	}

	if err != nil {
		return resp.Err("Error while getting bot count", err)
	}

	data := types.PagedResult[[]types.IndexBot]{
		Count:   count,
		Results: bots,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
