// Copyright (C) 2026 NodeByte LTD

package delete_pack

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
	"go.uber.org/zap"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Pack",
		Description: "Deletes a pack by URL. You *must* be the owner of the pack to delete packs. Returns 204 on success",
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
				Description: "The pack's URL",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	q := db.New(state.Pool)

	owner, err := q.GetPackOwner(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while checking pack owner [db fetch]", err, zap.String("id", id))
	}

	if owner != d.Auth.ID {
		return resp.Forbidden("You are not the owner of this pack")
	}

	err = q.DeletePack(d.Context, id)

	if err != nil {
		return resp.Err("Error while deleting pack [db exec]", err, zap.String("id", id))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
