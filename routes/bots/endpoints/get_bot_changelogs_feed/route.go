// Package get_bot_changelogs_feed implements GET /bots/@changelogs — "Get
// Bot Changelogs Feed".
//
// A chronological, sitewide feed of every bot's changelog entries, newest
// first. Public.
package get_bot_changelogs_feed

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
		Summary:     "Get Bot Changelogs Feed",
		Description: "A chronological, sitewide feed of every bot's changelog entries, newest first.",
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
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (int(pageNum) - 1) * perPage

	q := db.New(state.Pool)

	rows, err := q.GetBotChangelogsFeed(d.Context, db.GetBotChangelogsFeedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
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

	countRaw, err := q.CountBotChangelogs(d.Context)

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
