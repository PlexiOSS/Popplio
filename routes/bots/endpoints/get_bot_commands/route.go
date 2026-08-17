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
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
)

var (
	botCommandColsArr = db.GetCols(types.BotCommand{})
	botCommandCols    = strings.Join(botCommandColsArr, ",")
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

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+botCommandCols+" FROM bot_commands WHERE bot_id = $1 ORDER BY position ASC, created_at ASC",
		botId)

	if err != nil {
		return resp.Err("Failed to fetch bot commands [db fetch]", err)
	}

	commands, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.BotCommand])

	if err != nil {
		return resp.Err("Failed to fetch bot commands [collect]", err)
	}

	return uapi.HttpResponse{
		Json: types.BotCommandList{Commands: commands},
	}
}
