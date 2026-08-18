// Package get_flat_emojis implements GET /servers/@emojis/flat — "Get Flat
// Emojis".
//
// Unlike GET /servers/@emojis (paginated one server per page, each carrying
// its full emoji/sticker list), this unnests every opted-in server's emojis
// jsonb array and paginates at the individual-emoji level, so a browse page
// can show one flat, searchable list without server sections growing
// unbounded as more servers opt in.
package get_flat_emojis

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
		Summary:     "Get Flat Emojis",
		Description: "Gets a paginated, flat list of emojis across every server that has opted in to showing them",
		Resp:        types.PagedResult[[]types.FlatEmoji]{},
		RespName:    "PagedResultFlatEmoji",
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
			e->>'id' AS id, e->>'name' AS name, coalesce((e->>'animated')::boolean, false) AS animated, e->>'url' AS url
		FROM servers s
		CROSS JOIN LATERAL jsonb_array_elements(s.emojis) AS e
		WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'
		ORDER BY e->>'name' ASC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)

	if err != nil {
		return resp.Err("Error while querying flat emojis [db fetch]", err)
	}

	emojis, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.FlatEmoji])

	if err != nil {
		return resp.Err("Error while querying flat emojis [collect]", err)
	}

	var count uint64

	err = state.Pool.QueryRow(
		d.Context,
		`SELECT coalesce(SUM(jsonb_array_length(s.emojis)), 0)
		FROM servers s
		WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'`,
	).Scan(&count)

	if err != nil {
		return resp.Err("Error while counting flat emojis [db count]", err)
	}

	data := types.PagedResult[[]types.FlatEmoji]{
		Count:   count,
		Results: emojis,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
