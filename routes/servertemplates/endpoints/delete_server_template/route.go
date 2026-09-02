// Copyright (C) 2026 NodeByte LTD

package delete_server_template

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
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Server Template",
		Description: "Deletes a server template. You must be the owner to delete it. Returns 204 on success",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "id",
				Description: "The template's internal ID",
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

	owner, err := q.GetServerTemplateOwner(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while checking server template owner [db fetch]", err)
	}

	if owner != d.Auth.ID {
		return resp.Forbidden("You are not the owner of this server template")
	}

	if err := q.DeleteServerTemplate(d.Context, id); err != nil {
		return resp.Err("Error while deleting server template [db exec]", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
