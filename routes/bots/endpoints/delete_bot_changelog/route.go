// Package delete_bot_changelog implements DELETE
// /bots/{id}/changelogs/{changelog_id} — "Delete Bot Changelog".
//
// Removes a changelog/announcement entry. You must have 'Edit Bot
// Settings' in the team if the bot is in a team.
package delete_bot_changelog

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

	rowsAffected, err := db.New(state.Pool).DeleteBotChangelog(d.Context, db.DeleteBotChangelogParams{
		ID:    changelogId,
		BotID: botId,
	})

	if err != nil {
		return resp.Err("Failed to delete bot changelog", err, zap.String("botID", botId), zap.String("changelogID", changelogId))
	}

	if rowsAffected == 0 {
		return resp.NotFound("Changelog entry not found")
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
