// Copyright (C) 2026 NodeByte LTD

package sync_server_template

import (
	"errors"
	"net/http"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/servertemplates/assets"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/ratelimit"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Sync Server Template",
		Description: "Refreshes a server template's name, usage count, channels and roles from Discord. You must be the owner. Returns 204 on success, or 409 if the template no longer exists on Discord (it's then hidden from browse until it's back)",
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
	limit, err := ratelimit.Ratelimit{
		Expiry:      10 * time.Minute,
		MaxRequests: 5,
		Bucket:      "template_sync",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err)
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	id := chi.URLParam(r, "id")

	row, err := db.New(state.Pool).GetServerTemplateCode(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error while getting server template [db fetch]", err)
	}

	if row.Owner != d.Auth.ID {
		return resp.Forbidden("You are not the owner of this server template")
	}

	_, err = assets.Sync(d.Context, id, row.Code)

	if errors.Is(err, assets.ErrTemplateGone) {
		return resp.Status(http.StatusConflict, "This template no longer exists on Discord. It's hidden from browse until you create it again with the same code, or you can delete it.")
	}

	if err != nil {
		return resp.Status(http.StatusBadGateway, "Couldn't sync with Discord: "+err.Error())
	}

	return resp.NoContent()
}
