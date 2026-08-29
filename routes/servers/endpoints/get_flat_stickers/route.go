// Package get_flat_stickers implements GET /servers/@stickers/flat — "Get
// Flat Stickers". Sticker counterpart of get_flat_emojis; see that package
// for why this unnests rather than paginating per-server.
package get_flat_stickers

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
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

	q := db.New(state.Pool)

	rows, err := q.GetFlatStickers(d.Context, db.GetFlatStickersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying flat stickers [db fetch]", err)
	}

	stickers := make([]types.FlatSticker, len(rows))
	for i, row := range rows {
		stickers[i] = types.FlatSticker{
			ServerID:     row.ServerID,
			ServerName:   row.ServerName,
			ServerAvatar: row.ServerAvatar,
			ID:           row.ID,
			Name:         row.Name,
			Format:       row.Format,
			URL:          row.Url,
		}
	}

	countRaw, err := q.CountFlatStickers(d.Context)

	if err != nil {
		return resp.Err("Error while counting flat stickers [db count]", err)
	}

	count := uint64(countRaw)

	data := types.PagedResult[[]types.FlatSticker]{
		Count:   count,
		Results: stickers,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
