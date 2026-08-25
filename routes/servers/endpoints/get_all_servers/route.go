package get_all_servers

import (
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/pagination"
	"popplio/routes/servers/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

var (
	indexServerColsArr = dbutil.GetCols(types.IndexServer{})
	indexServerCols    = strings.Join(indexServerColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Servers",
		Description: "Gets all servers on the list. Returns a set of paginated ``IndexServer`` objects",
		Resp:        types.PagedResult[[]types.IndexServer]{},
		RespName:    "PagedResultIndexServer",
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
				Description: "Sort order. Omit for newest-first; \"trending\" ranks by net votes in the last 7 days instead, and only returns servers with at least one vote in that window.",
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
				WHERE target_type = 'server' AND void = false AND created_at > now() - interval '`+trendingWindow+`'
				GROUP BY target_id
			)
			SELECT `+indexServerCols+` FROM servers
			WHERE (type = 'approved' OR type = 'certified') AND state = 'public' AND server_id IN (SELECT target_id FROM scored)
			ORDER BY (SELECT score FROM scored WHERE scored.target_id = servers.server_id) DESC
			LIMIT $1 OFFSET $2`, limit, offset)
	} else {

		rows, err = state.Pool.Query(d.Context, "SELECT "+indexServerCols+" FROM servers WHERE (type = 'approved' OR type = 'certified') AND state = 'public' ORDER BY (boosted_until IS NOT NULL AND boosted_until > NOW()) DESC, created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	}

	if err != nil {
		return resp.Err("Failed to query servers [db query]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	servers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexServer])

	if err != nil {
		return resp.Err("Failed to query servers [collect]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	if err := assets.ResolveIndexServers(d.Context, servers); err != nil {
		return resp.ErrBody("Error resolving indexserver", "An error occurred while resolving index server.", err)
	}

	var count uint64

	if trending {
		err = state.Pool.QueryRow(d.Context, `
			SELECT COUNT(*) FROM servers
			WHERE (type = 'approved' OR type = 'certified') AND state = 'public' AND server_id IN (
				SELECT target_id FROM entity_votes
				WHERE target_type = 'server' AND void = false AND created_at > now() - interval '`+trendingWindow+`'
				GROUP BY target_id
			)`).Scan(&count)
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers").Scan(&count)
	}

	if err != nil {
		return resp.Err("Failed to query servers [db count]", err, zap.Uint64("page", pageNum), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	data := types.PagedResult[[]types.IndexServer]{
		Count:   count,
		Results: servers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
