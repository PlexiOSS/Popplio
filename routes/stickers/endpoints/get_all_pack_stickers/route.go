// Copyright (C) 2026 NodeByte LTD

package get_all_pack_stickers

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

const perPage = 24

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Pack Stickers",
		Description: "Gets every pack sticker across every sticker pack, newest first. This endpoint is paginated.",
		Resp:        types.PagedResult[[]types.FlatPackSticker]{},
		RespName:    "PagedResultFlatPackSticker",
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

	rows, err := q.GetAllPackStickersPaged(d.Context, db.GetAllPackStickersPagedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying pack stickers [db fetch]", err)
	}

	stickers := make([]types.FlatPackSticker, len(rows))
	for i, row := range rows {
		stickers[i] = types.FlatPackSticker{
			ID:        row.ID,
			Name:      row.Name,
			Animated:  row.Animated,
			Downloads: int(row.Downloads),
			CreatedAt: row.CreatedAt.Time,
			PackURL:   row.PackUrl,
			PackName:  row.PackName,
		}
	}

	count, err := q.CountPackStickers(d.Context)

	if err != nil {
		return resp.Err("Error while counting pack stickers [db count]", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.FlatPackSticker]{
			Count:   uint64(count),
			PerPage: perPage,
			Results: stickers,
		},
	}
}
