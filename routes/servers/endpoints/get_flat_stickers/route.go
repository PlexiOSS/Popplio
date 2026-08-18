// Package get_flat_stickers implements GET /servers/@stickers/flat — "Get
// Flat Stickers". Sticker counterpart of get_flat_emojis; see that package
// for why this unnests rather than paginating per-server.
package get_flat_stickers

import (
	"net/http"
	"popplio/api/resp"

	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const perPage = 60

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Flat Stickers",
		Description: "Gets a paginated, flat list of stickers across every server that has opted in to showing them",
		Resp:        types.PagedResult[[]types.FlatSticker]{},
		RespName:    "PagedResultFlatSticker",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
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

	rows, err := state.Pool.Query(
		d.Context,
		`SELECT s.server_id AS server_id, s.name AS server_name, s.avatar AS server_avatar,
			e->>'id' AS id, e->>'name' AS name, e->>'format' AS format, e->>'url' AS url
		FROM servers s
		CROSS JOIN LATERAL jsonb_array_elements(s.stickers) AS e
		WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'
		ORDER BY e->>'name' ASC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)

	if err != nil {
		return resp.Err("Error while querying flat stickers [db fetch]", err)
	}

	stickers, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.FlatSticker])

	if err != nil {
		return resp.Err("Error while querying flat stickers [collect]", err)
	}

	var count uint64

	err = state.Pool.QueryRow(
		d.Context,
		`SELECT coalesce(SUM(jsonb_array_length(s.stickers)), 0)
		FROM servers s
		WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'`,
	).Scan(&count)

	if err != nil {
		return resp.Err("Error while counting flat stickers [db count]", err)
	}

	data := types.PagedResult[[]types.FlatSticker]{
		Count:   count,
		Results: stickers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
