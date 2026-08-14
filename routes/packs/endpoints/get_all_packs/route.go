// Package get_all_packs implements GET /packs/@all — "Get All Packs".
//
// Gets all packs on the list. This endpoint is paginated.
package get_all_packs

import (
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	"popplio/routes/packs/assets"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const perPage = 12

var (
	packColArr = db.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Packs",
		Description: "Gets all packs on the list. This endpoint is paginated.",
		Resp:        types.PagedResult[[]types.BotPack]{},
		RespName:    "PagedResultIndexBotPack",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "pack_type",
				Description: "Filter to only packs of this type (bot, server, or emoji). Omit to return every type.",
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

	packType := r.URL.Query().Get("pack_type")

	var rows pgx.Rows

	if packType != "" {
		if packType != types.PackTypeBot && packType != types.PackTypeServer && packType != types.PackTypeEmoji {
			return resp.BadRequest("pack_type must be one of bot, server, or emoji")
		}

		rows, err = state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE pack_type = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3", packType, limit, offset)
	} else {
		rows, err = state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs ORDER BY created_at DESC LIMIT $1 OFFSET $2", limit, offset)
	}

	if err != nil {
		return resp.Err("Error while querying packs [db fetch]", err)
	}

	packs, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.BotPack])

	if err != nil {
		return resp.Err("Error while querying packs [collect]", err)
	}

	for i := range packs {
		err = assets.ResolveBotPack(d.Context, &packs[i])

		if err != nil {
			return resp.ErrDetail("Error resolving bot pack", err, zap.String("url", packs[i].URL))
		}
	}

	var count uint64

	if packType != "" {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs WHERE pack_type = $1", packType).Scan(&count)
	} else {
		err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM packs").Scan(&count)
	}

	if err != nil {
		return resp.Err("Error while querying packs [db count]", err)
	}

	data := types.PagedResult[[]types.BotPack]{
		Count:   count,
		PerPage: perPage,
		Results: packs,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
