// Copyright (C) 2026 NodeByte LTD

package get_all_pack_sounds

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
		Summary:     "Get All Pack Sounds",
		Description: "Gets every pack sound across every sound pack, newest first. This endpoint is paginated.",
		Resp:        types.PagedResult[[]types.FlatPackSound]{},
		RespName:    "PagedResultFlatPackSound",
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

	rows, err := q.GetAllPackSoundsPaged(d.Context, db.GetAllPackSoundsPagedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying pack sounds [db fetch]", err)
	}

	sounds := make([]types.FlatPackSound, len(rows))
	for i, row := range rows {
		sounds[i] = types.FlatPackSound{
			ID:         row.ID,
			Name:       row.Name,
			DurationMs: int(row.DurationMs),
			Downloads:  int(row.Downloads),
			CreatedAt:  row.CreatedAt.Time,
			PackURL:    row.PackUrl,
			PackName:   row.PackName,
		}
	}

	count, err := q.CountPackSounds(d.Context)

	if err != nil {
		return resp.Err("Error while counting pack sounds [db count]", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.FlatPackSound]{
			Count:   uint64(count),
			PerPage: perPage,
			Results: sounds,
		},
	}
}
