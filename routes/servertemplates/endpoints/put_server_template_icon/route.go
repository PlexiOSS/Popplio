// Copyright (C) 2026 NodeByte LTD

package put_server_template_icon

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/entityassets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

type PutServerTemplateIcon struct {
	Version *string `json:"version" description:"The uploaded icon's version, or null to remove the icon"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Set Server Template Icon",
		Description: "Sets or removes a server template's custom icon. The icon itself is uploaded through the website first. You must be the owner. Returns 204 on success",
		Req:         PutServerTemplateIcon{},
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

	var payload PutServerTemplateIcon

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	version := pgtype.Text{}

	if payload.Version != nil {
		if !entityassets.ValidVersion(*payload.Version) {
			return resp.BadRequest("version must be 16 to 64 lowercase hex characters")
		}

		version = pgtype.Text{String: *payload.Version, Valid: true}
	}

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

	if err := q.SetServerTemplateIcon(d.Context, db.SetServerTemplateIconParams{ID: id, IconVersion: version}); err != nil {
		return resp.Err("Error while saving server template icon", err)
	}

	return resp.NoContent()
}
