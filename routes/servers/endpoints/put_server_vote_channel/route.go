// Copyright (C) 2026 NodeByte LTD

package put_server_vote_channel

import (
	"net/http"
	"slices"

	"popplio/api/resp"
	"popplio/db"
	"popplio/servervotes"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.ServerVoteChannelUpdate{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Set Server Vote Messages",
		Description: "Sets the channel the Infernoplex bot posts new votes in, or turns vote messages off with a null channel_id. The bot posts a test message first and the change is only saved if that works. Needs 'Edit Server Settings'.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Server ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.ServerVoteChannelUpdate{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	serverID := chi.URLParam(r, "id")

	var payload types.ServerVoteChannelUpdate

	if hresp, ok := uapi.MarshalReq(r, &payload); !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		return uapi.ValidatorErrorResponse(compiledMessages, err.(validator.ValidationErrors))
	}

	channel := pgtype.Text{}

	if payload.ChannelID != nil && *payload.ChannelID != "" {
		channels, err := servervotes.Channels(serverID)

		if err != nil {
			return resp.BadRequest("The Infernoplex bot can't see this server. Add it to the server first.")
		}

		if !slices.ContainsFunc(channels, func(c types.ServerVoteChannelOption) bool { return c.ID == *payload.ChannelID }) {
			return resp.BadRequest("That channel isn't a text or announcement channel in this server")
		}

		if err := servervotes.SendTest(*payload.ChannelID); err != nil {
			return resp.BadRequest("The Infernoplex bot couldn't post in that channel. Give it permission to view the channel, send messages and embed links.")
		}

		channel = pgtype.Text{String: *payload.ChannelID, Valid: true}
	}

	if err := db.New(state.Pool).SetServerVoteChannel(d.Context, db.SetServerVoteChannelParams{
		ChannelID: channel,
		ServerID:  serverID,
	}); err != nil {
		return resp.Err("Failed to save vote messages", err)
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
