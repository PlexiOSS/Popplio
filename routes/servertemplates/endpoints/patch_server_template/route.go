// Copyright (C) 2026 NodeByte LTD

package patch_server_template

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var compiledMessages = uapi.CompileValidationErrors(PatchServerTemplate{})

// PatchServerTemplate deliberately excludes name (pulled from Discord's own
// template metadata, not independently settable) and channels/roles
// (re-syncing those from Discord is a separate, larger feature).
type PatchServerTemplate struct {
	Short string   `json:"short" validate:"required,min=10,max=150,noxss" msg:"Description must be between 10 and 150 characters"`
	Tags  []string `json:"tags" validate:"required,unique,min=1,max=5,dive,min=3,max=30,notblank,nonvulgar" msg:"There must be between 1 and 5 tags without duplicates" amsg:"Each tag must be between 3 and 30 characters and alphabetic"`
	NSFW  bool     `json:"nsfw"`
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update Server Template",
		Description: "Updates a server template's description, tags, and NSFW flag. You must be the owner. Returns 204 on success",
		Req:         PatchServerTemplate{},
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

	var payload PatchServerTemplate

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errs := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errs)
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

	if err := q.UpdateServerTemplate(d.Context, db.UpdateServerTemplateParams{
		ID:    id,
		Short: payload.Short,
		Tags:  payload.Tags,
		Nsfw:  payload.NSFW,
	}); err != nil {
		return resp.Err("Error while updating server template [db exec]", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
