// Package get_bot_changelogs_feed implements GET /bots/@changelogs — "Get
// Bot Changelogs Feed".
//
// A chronological, sitewide feed of every bot's changelog entries, newest
// first. Public.
package get_bot_changelogs_feed

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
		Summary:     "Get Bot Changelogs Feed",
		Description: "A chronological, sitewide feed of changelog entries from approved and certified bots, newest first.",
		Resp:        types.PagedResult[[]types.BotChangelogFeedEntry]{},
		RespName:    "PagedResultBotChangelogFeedEntry",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "q",
				Description: "Search the title, content and bot name (up to 100 characters)",
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
				Name:        "bot",
				Description: "Only this bot's entries (bot ID)",
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
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (int(pageNum) - 1) * perPage

	query := r.URL.Query()

	search := strings.TrimSpace(query.Get("q"))
	if len([]rune(search)) > 100 {
		return resp.BadRequest("Search query must be 100 characters or fewer")
	}

	botID := strings.TrimSpace(query.Get("bot"))
	if botID != "" && !snowflakeRe.MatchString(botID) {
		return resp.BadRequest("bot must be a bot ID")
	}

	tag := strings.TrimSpace(query.Get("tag"))
	if len(tag) > 64 {
		return resp.BadRequest("tag is too long")
	}

	filterBot := pgtype.Text{String: botID, Valid: botID != ""}
	filterTag := pgtype.Text{String: tag, Valid: tag != ""}
	filterPattern := pgtype.Text{String: "%" + likeEscaper.Replace(search) + "%", Valid: search != ""}

	q := db.New(state.Pool)

	rows, err := q.GetBotChangelogsFeed(d.Context, db.GetBotChangelogsFeedParams{
		BotID:   filterBot,
		Tag:     filterTag,
		Pattern: filterPattern,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})

	if err != nil {
		return resp.Err("Error while getting bot changelogs feed [db fetch]", err)
	}

	entries := make([]types.BotChangelogFeedEntry, len(rows))
	for i, row := range rows {
		entries[i] = types.BotChangelogFeedEntry{
			ID:        row.ID,
			BotID:     row.BotID,
			Title:     row.Title,
			Content:   row.Content,
			Version:   row.Version,
			CreatedAt: row.CreatedAt.Time,
		}
	}

	if err := assets.ResolveBotChangelogFeedEntries(d.Context, entries); err != nil {
		return resp.ErrBody("Error resolving bot changelog feed", "An error occurred while resolving the bot changelog feed.", err)
	}

	countRaw, err := q.CountBotChangelogsFeed(d.Context, db.CountBotChangelogsFeedParams{
		BotID:   filterBot,
		Tag:     filterTag,
		Pattern: filterPattern,
	})

	if err != nil {
		return resp.Err("Error while getting bot changelog count", err)
	}

	data := types.PagedResult[[]types.BotChangelogFeedEntry]{
		Count:   uint64(countRaw),
		Results: entries,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
