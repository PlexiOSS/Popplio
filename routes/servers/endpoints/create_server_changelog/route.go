// Copyright (C) 2026 NodeByte LTD

package create_server_changelog

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateServerChangelog{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Server Changelog",
		Description: "Posts an update for a server, shown on its page and in the Feed. Needs 'Edit Server Settings' in the server's team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.CreateServerChangelog{},
		Resp: types.ServerChangelog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverID := chi.URLParam(r, "id")

	var payload types.CreateServerChangelog

	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	row, err := db.New(state.Pool).InsertServerChangelog(d.Context, db.InsertServerChangelogParams{
		ServerID:  serverID,
		Title:     payload.Title,
		Content:   payload.Content,
		Version:   payload.Version,
		CreatedBy: d.Auth.ID,
	})

	if err != nil {
		return resp.Err("Failed to create server changelog", err, zap.String("serverID", serverID))
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json: types.ServerChangelog{
			ID:        row.ID,
			Title:     row.Title,
			Content:   row.Content,
			Version:   row.Version,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt.Time,
		},
	}
}
