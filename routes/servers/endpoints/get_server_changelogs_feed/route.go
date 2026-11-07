// Copyright (C) 2026 NodeByte LTD

package get_server_changelogs_feed

import (
	"net/http"
	"regexp"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 20

var (
	snowflakeRe = regexp.MustCompile(`^[0-9]{16,20}$`)
	likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Server Changelogs Feed",
		Description: "A sitewide feed of updates from approved and certified servers, newest first.",
		Resp:        types.PagedResult[[]types.ServerChangelogFeedEntry]{},
		RespName:    "PagedResultServerChangelogFeedEntry",
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
				Description: "Search the title, content and server name (up to 100 characters)",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tag",
				Description: "Only servers with this tag (case-insensitive)",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "server",
				Description: "Only this server's updates (server ID)",
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

	query := r.URL.Query()

	search := strings.TrimSpace(query.Get("q"))

	if len([]rune(search)) > 100 {
		return resp.BadRequest("Search query must be 100 characters or fewer")
	}

	serverID := strings.TrimSpace(query.Get("server"))

	if serverID != "" && !snowflakeRe.MatchString(serverID) {
		return resp.BadRequest("server must be a server ID")
	}

	tag := strings.TrimSpace(query.Get("tag"))

	if len(tag) > 64 {
		return resp.BadRequest("tag is too long")
	}

	filterServer := pgtype.Text{String: serverID, Valid: serverID != ""}
	filterTag := pgtype.Text{String: tag, Valid: tag != ""}
	filterPattern := pgtype.Text{String: "%" + likeEscaper.Replace(search) + "%", Valid: search != ""}

	q := db.New(state.Pool)

	rows, err := q.GetServerChangelogsFeed(d.Context, db.GetServerChangelogsFeedParams{
		ServerID: filterServer,
		Tag:      filterTag,
		Pattern:  filterPattern,
		Limit:    perPage,
		Offset:   int32((int(pageNum) - 1) * perPage),
	})

	if err != nil {
		return resp.Err("Error while getting server changelogs feed", err)
	}

	entries := make([]types.ServerChangelogFeedEntry, len(rows))

	for i, row := range rows {
		entries[i] = types.ServerChangelogFeedEntry{
			ID:        row.ID,
			ServerID:  row.ServerID,
			Name:      row.ServerID,
			Title:     row.Title,
			Content:   row.Content,
			Version:   row.Version,
			CreatedAt: row.CreatedAt.Time,
		}

		if info, err := votes.GetEntityInfo(d.Context, state.Pool, row.ServerID, "server"); err == nil {
			entries[i].Name = info.Name
			entries[i].Avatar = info.Avatar
		}
	}

	count, err := q.CountServerChangelogsFeed(d.Context, db.CountServerChangelogsFeedParams{
		ServerID: filterServer,
		Tag:      filterTag,
		Pattern:  filterPattern,
	})

	if err != nil {
		return resp.Err("Error while getting server changelog count", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.ServerChangelogFeedEntry]{
			Count:   uint64(count),
			Results: entries,
			PerPage: perPage,
		},
	}
}
