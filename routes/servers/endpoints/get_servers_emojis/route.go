package get_servers_emojis

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

const perPage = 12

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

	q := db.New(state.Pool)

	rows, err := q.GetServerEmojiPreviews(d.Context, db.GetServerEmojiPreviewsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying servers emojis [db fetch]", err)
	}

	previews := make([]types.ServerEmojiPreview, len(rows))
	for i, row := range rows {
		previews[i] = types.ServerEmojiPreview{
			ServerID: row.ServerID,
			Name:     row.Name,
			Avatar:   row.Avatar,
			Emojis:   row.Emojis,
			Stickers: row.Stickers,
		}
	}

	countRaw, err := q.CountServerEmojiPreviews(d.Context)

	if err != nil {
		return resp.Err("Error while counting servers emojis [db count]", err)
	}

	count := uint64(countRaw)

	data := types.PagedResult[[]types.ServerEmojiPreview]{
		Count:   count,
		Results: previews,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
