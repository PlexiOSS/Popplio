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

	q := db.New(state.Pool)

	rows, err := q.GetFlatEmojis(d.Context, db.GetFlatEmojisParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying flat emojis [db fetch]", err)
	}

	emojis := make([]types.FlatEmoji, len(rows))
	for i, row := range rows {
		emojis[i] = types.FlatEmoji{
			ServerID:     row.ServerID,
			ServerName:   row.ServerName,
			ServerAvatar: row.ServerAvatar,
			ID:           row.ID,
			Name:         row.Name,
			Animated:     row.Animated,
			URL:          row.Url,
		}
	}

	countRaw, err := q.CountFlatEmojis(d.Context)

	if err != nil {
		return resp.Err("Error while counting flat emojis [db count]", err)
	}

	count := uint64(countRaw)

	data := types.PagedResult[[]types.FlatEmoji]{
		Count:   count,
		Results: emojis,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
