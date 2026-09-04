// Package search_bot_commands implements GET /bots/@commands — "Search Bot
// Commands".
//
// Cross-bot command search -- "who has a /giveaway command." Public.
package search_bot_commands

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 20

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Search Bot Commands",
		Description: "Cross-bot command search -- finds every bot with a matching command name.",
		Resp:        types.PagedResult[[]types.BotCommandSearchResult]{},
		RespName:    "PagedResultBotCommandSearchResult",
		Params: []docs.Parameter{
			{
				Name:        "query",
				Description: "Command name (or partial name) to search for",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	query := r.URL.Query().Get("query")

	if query == "" {
		return resp.BadRequest("query must be specified")
	}

	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (int(pageNum) - 1) * perPage

	pattern := "%" + query + "%"

	q := db.New(state.Pool)

	rows, err := q.SearchBotCommands(d.Context, db.SearchBotCommandsParams{
		Name:   pattern,
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error while searching bot commands [db fetch]", err)
	}

	results := make([]types.BotCommandSearchResult, len(rows))
	for i, row := range rows {
		results[i] = types.BotCommandSearchResult{
			ID:          row.ID,
			BotID:       row.BotID,
			Name:        row.Name,
			Description: row.Description,
			Usage:       row.Usage,
			Category:    row.Category,
		}
	}

	if err := assets.ResolveBotCommandSearchResults(d.Context, results); err != nil {
		return resp.ErrBody("Error resolving bot command search results", "An error occurred while resolving bot command search results.", err)
	}

	countRaw, err := q.CountBotCommandsSearch(d.Context, pattern)

	if err != nil {
		return resp.Err("Error while counting bot commands [db count]", err)
	}

	data := types.PagedResult[[]types.BotCommandSearchResult]{
		Count:   uint64(countRaw),
		Results: results,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
