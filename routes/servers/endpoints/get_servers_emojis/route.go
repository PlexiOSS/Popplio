package get_servers_emojis

import (
	"net/http"
	"popplio/api/resp"
	"strings"

	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"
)

const perPage = 12

var (
	previewColsArr = db.GetCols(types.ServerEmojiPreview{})
	previewCols    = strings.Join(previewColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Servers Emojis",
		Description: "Gets a paginated list of servers that have opted in to showing their emojis/stickers",
		Resp:        types.PagedResult[[]types.ServerEmojiPreview]{},
		RespName:    "PagedResultServerEmojiPreview",
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
		"SELECT "+previewCols+" FROM servers WHERE show_emojis = true AND (type = 'approved' OR type = 'certified') AND state = 'public' ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset,
	)

	if err != nil {
		return resp.Err("Error while querying servers emojis [db fetch]", err)
	}

	previews, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ServerEmojiPreview])

	if err != nil {
		return resp.Err("Error while querying servers emojis [collect]", err)
	}

	var count uint64

	err = state.Pool.QueryRow(d.Context, "SELECT COUNT(*) FROM servers WHERE show_emojis = true AND (type = 'approved' OR type = 'certified') AND state = 'public'").Scan(&count)

	if err != nil {
		return resp.Err("Error while counting servers emojis [db count]", err)
	}

	data := types.PagedResult[[]types.ServerEmojiPreview]{
		Count:   count,
		Results: previews,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
