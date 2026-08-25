// Package create_bot_changelog implements POST /bots/{id}/changelogs —
// "Create Bot Changelog".
//
// Posts a new changelog/announcement entry for a bot. You must have 'Edit
// Bot Settings' in the team if the bot is in a team.
package create_bot_changelog

import (
	"net/http"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateBotChangelog{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Bot Changelog",
		Description: "Posts a new changelog/announcement entry for a bot. You must have 'Edit Bot Settings' in the team if the bot is in a team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.CreateBotChangelog{},
		Resp: types.BotChangelog{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "id")

	var payload types.CreateBotChangelog

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	if err := state.Validator.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	var entry types.BotChangelog

	err := state.Pool.QueryRow(d.Context,
		"INSERT INTO bot_changelogs (bot_id, title, content, version, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, content, version, created_by, created_at",
		botId, payload.Title, payload.Content, payload.Version, d.Auth.ID,
	).Scan(&entry.ID, &entry.Title, &entry.Content, &entry.Version, &entry.CreatedBy, &entry.CreatedAt)

	if err != nil {
		return resp.Err("Failed to create bot changelog", err, zap.String("botID", botId))
	}

	return uapi.HttpResponse{
		Status: http.StatusCreated,
		Json:   entry,
	}
}
