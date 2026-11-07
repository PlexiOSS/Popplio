// Copyright (C) 2026 NodeByte LTD

package delete_server_changelog

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
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Server Changelog",
		Description: "Removes a server update. Needs 'Edit Server Settings' in the server's team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "changelog_id",
				Description: "Changelog entry ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverID := chi.URLParam(r, "id")
	changelogID := chi.URLParam(r, "changelog_id")

	rows, err := db.New(state.Pool).DeleteServerChangelog(d.Context, db.DeleteServerChangelogParams{
		ID:       changelogID,
		ServerID: serverID,
	})

	if err != nil {
		return resp.Err("Failed to delete server changelog", err, zap.String("serverID", serverID), zap.String("changelogID", changelogID))
	}

	if rows == 0 {
		return resp.NotFound("Changelog entry not found")
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
