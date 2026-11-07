// Copyright (C) 2026 NodeByte LTD

package get_server_changelogs

import (
	"net/http"

	"popplio/api"
	"popplio/api/resp"
	"popplio/db"
	"popplio/listing"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Changelogs",
		Description: "Gets the updates a server's owners have posted, newest first.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ServerChangelogList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverID := chi.URLParam(r, "id")

	visible, err := listing.Visible(d.Context, d.Auth, api.TargetTypeServer, serverID)

	if err != nil {
		return resp.Err("Failed to check listing status", err)
	}

	if !visible {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	rows, err := db.New(state.Pool).GetServerChangelogs(d.Context, serverID)

	if err != nil {
		return resp.Err("Failed to fetch server changelogs", err)
	}

	changelogs := make([]types.ServerChangelog, len(rows))

	for i, row := range rows {
		changelogs[i] = types.ServerChangelog{
			ID:        row.ID,
			Title:     row.Title,
			Content:   row.Content,
			Version:   row.Version,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{Json: types.ServerChangelogList{Changelogs: changelogs}}
}
