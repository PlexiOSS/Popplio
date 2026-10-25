// Package search_bot_commands implements GET /bots/@commands — "Search Bot
// Commands".
//
// Cross-bot command search -- "who has a /giveaway command." Public.
package search_bot_commands

import (
	"net/http"
	"regexp"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 20

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Search Bot Commands",
		Description: "Cross-bot command search across approved and certified bots. Matches command names and descriptions; at least one of query, bot or tag is required.",
		Resp:        types.PagedResult[[]types.BotCommandSearchResult]{},
		RespName:    "PagedResultBotCommandSearchResult",
		Params: []docs.Parameter{
			{
				Name:        "query",
				Description: "Text to find in command names and descriptions (up to 100 characters, matched literally)",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "bot",
				Description: "Only this bot's commands (bot ID)",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tag",
				Description: "Only bots with this tag (case-insensitive)",
				Required:    false,
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

var snowflakeRe = regexp.MustCompile(`^[0-9]{16,20}$`)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	params := r.URL.Query()
	query := strings.TrimSpace(params.Get("query"))
	botID := strings.TrimSpace(params.Get("bot"))
	tag := strings.TrimSpace(params.Get("tag"))

	if query == "" && botID == "" && tag == "" {
		return resp.BadRequest("query, bot or tag must be specified")
	}

	if len([]rune(query)) > 100 {
		return resp.BadRequest("query must be 100 characters or fewer")
	}

	if botID != "" && !snowflakeRe.MatchString(botID) {
		return resp.BadRequest("bot must be a bot ID")
	}

	if len(tag) > 64 {
		return resp.BadRequest("tag is too long")
	}

	filterPattern := pgtype.Text{String: "%" + likeEscaper.Replace(query) + "%", Valid: query != ""}
	filterBot := pgtype.Text{String: botID, Valid: botID != ""}
	filterTag := pgtype.Text{String: tag, Valid: tag != ""}

	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (int(pageNum) - 1) * perPage

	q := db.New(state.Pool)

	rows, err := q.SearchBotCommands(d.Context, db.SearchBotCommandsParams{
		Pattern: filterPattern,
		BotID:   filterBot,
		Tag:     filterTag,
		Limit:   int32(limit),
		Offset:  int32(offset),
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

	countRaw, err := q.CountBotCommandsSearch(d.Context, db.CountBotCommandsSearchParams{
		Pattern: filterPattern,
		BotID:   filterBot,
		Tag:     filterTag,
	})

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
