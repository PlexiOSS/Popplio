// Package get_bot_changelogs implements GET /bots/{id}/changelogs — "Get
// Bot Changelogs".
//
// Gets the changelog/announcement entries a bot's owner/team has posted.
// Public.
package get_bot_changelogs

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Bot Changelogs",
		Description: "Gets the changelog/announcement entries a bot's owner/team has posted.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotChangelogList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "id")

	rows, err := db.New(state.Pool).GetBotChangelogs(d.Context, botId)

	if err != nil {
		return resp.Err("Failed to fetch bot changelogs [db fetch]", err)
	}

	changelogs := make([]types.BotChangelog, len(rows))
	for i, row := range rows {
		changelogs[i] = types.BotChangelog{
			ID:        row.ID,
			Title:     row.Title,
			Content:   row.Content,
			Version:   row.Version,
			CreatedBy: row.CreatedBy,
			CreatedAt: row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Json: types.BotChangelogList{Changelogs: changelogs},
	}
}
