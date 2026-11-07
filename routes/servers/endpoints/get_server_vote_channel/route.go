// Copyright (C) 2026 NodeByte LTD

package get_server_vote_channel

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/servervotes"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Vote Messages",
		Description: "Gets the channel the Infernoplex bot posts new votes in, and the server's channels to choose from. Needs 'Edit Server Settings'.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ServerVoteChannel{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverID := chi.URLParam(r, "id")

	current, err := db.New(state.Pool).GetServerVoteChannel(d.Context, serverID)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to load vote messages", err)
	}

	out := types.ServerVoteChannel{Channels: []types.ServerVoteChannelOption{}}

	if current.Valid {
		out.ChannelID = &current.String
	}

	channels, err := servervotes.Channels(serverID)

	if err == nil {
		out.Channels = channels
		out.Available = true
	}

	return uapi.HttpResponse{Json: out}
}
