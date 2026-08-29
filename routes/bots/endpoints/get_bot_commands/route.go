// Package get_bot_commands implements GET /bots/{id}/commands — "Get Bot
// Commands".
//
// Gets the commands a bot's owner/team has documented for it. Public.
package get_bot_commands

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
		Summary:     "Get Bot Commands",
		Description: "Gets the commands a bot's owner/team has documented for it.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Bot ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotCommandList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	botId := chi.URLParam(r, "id")

	rows, err := db.New(state.Pool).GetBotCommands(d.Context, botId)

	if err != nil {
		return resp.Err("Failed to fetch bot commands [db fetch]", err)
	}

	commands := make([]types.BotCommand, len(rows))
	for i, row := range rows {
		commands[i] = types.BotCommand{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Usage:       row.Usage,
			Category:    row.Category,
			Position:    int(row.Position),
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Json: types.BotCommandList{Commands: commands},
	}
}
