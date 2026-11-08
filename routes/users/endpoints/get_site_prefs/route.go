// Copyright (C) 2026 NodeByte LTD

package get_site_prefs

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Site Preferences",
		Description: "Returns the user's saved website customization settings, so they follow them across devices. prefs is empty and updated_at is null if they never saved any. Only the user themselves can read them.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.SitePrefs{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	row, err := db.New(state.Pool).GetSitePrefs(d.Context, chi.URLParam(r, "id"))

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.HttpResponse{Json: types.SitePrefs{Prefs: map[string]any{}}}
	}

	if err != nil {
		return resp.Err("Failed to load site preferences", err)
	}

	t := row.UpdatedAt.Time

	return uapi.HttpResponse{Json: types.SitePrefs{Prefs: row.Prefs, UpdatedAt: &t}}
}
