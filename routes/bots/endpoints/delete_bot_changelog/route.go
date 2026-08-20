// Package delete_bot_changelog implements DELETE
// /bots/{id}/changelogs/{changelog_id} — "Delete Bot Changelog".
//
// Removes a changelog/announcement entry. You must have 'Edit Bot
// Settings' in the team if the bot is in a team.
package delete_bot_changelog

import (
	"net/http"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Bot Changelog",
		Description: "Removes a changelog/announcement entry. You must have 'Edit Bot Settings' in the team if the bot is in a team.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
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
	botId := chi.URLParam(r, "id")
	changelogId := chi.URLParam(r, "changelog_id")

	tag, err := state.Pool.Exec(d.Context,
		"DELETE FROM bot_changelogs WHERE id = $1 AND bot_id = $2",
		changelogId, botId)

	if err != nil {
		return resp.Err("Failed to delete bot changelog", err, zap.String("botID", botId), zap.String("changelogID", changelogId))
	}

	if tag.RowsAffected() == 0 {
		return resp.NotFound("Changelog entry not found")
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
