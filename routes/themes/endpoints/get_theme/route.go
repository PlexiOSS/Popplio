// Copyright (C) 2026 NodeByte LTD

package get_theme

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Theme",
		Description: "Gets a single theme by its own ID, with its owner resolved",
		Resp:        types.Theme{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The theme's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	q := db.New(state.Pool)

	row, err := q.GetThemeByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting theme [db fetch]", err)
	}

	owner, err := dovewing.GetUser(d.Context, row.Owner, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Error querying dovewing for owner user", err)
	}

	return uapi.HttpResponse{
		Json: types.Theme{
			ID:             row.ID,
			Name:           row.Name,
			PrimaryColor:   row.PrimaryColor,
			SecondaryColor: row.SecondaryColor,
			Tags:           row.Tags,
			CreatedAt:      row.CreatedAt.Time,
			ResolvedOwner:  owner,
		},
	}
}
