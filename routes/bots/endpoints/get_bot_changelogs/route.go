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
	"strings"

	docs "github.com/infinitybotlist/eureka/doclib"
	"github.com/infinitybotlist/eureka/uapi"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
)

var (
	botChangelogColsArr = db.GetCols(types.BotChangelog{})
	botChangelogCols    = strings.Join(botChangelogColsArr, ",")
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

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+botChangelogCols+" FROM bot_changelogs WHERE bot_id = $1 ORDER BY created_at DESC",
		botId)

	if err != nil {
		return resp.Err("Failed to fetch bot changelogs [db fetch]", err)
	}

	changelogs, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.BotChangelog])

	if err != nil {
		return resp.Err("Failed to fetch bot changelogs [collect]", err)
	}

	return uapi.HttpResponse{
		Json: types.BotChangelogList{Changelogs: changelogs},
	}
}
