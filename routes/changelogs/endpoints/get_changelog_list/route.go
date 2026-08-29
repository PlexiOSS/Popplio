// Package get_changelog_list implements GET /changelogs/@all — "Get
// Changelog List".
//
// Gets published changelog entries for Popplio and/or Omniplex, newest
// first.
package get_changelog_list

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Changelog List",
		Description: "Gets published changelog entries for Popplio, Omniplex, and/or Keel, newest first.",
		Params: []docs.Parameter{
			{
				Name:        "project",
				Description: "Filter to one project: 'popplio', 'omniplex', or 'keel'. Omit for all.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ChangelogList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var project pgtype.Text

	if v := r.URL.Query().Get("project"); v != "" {
		project = pgtype.Text{String: v, Valid: true}
	}

	rows, err := db.New(state.Pool).GetChangelogList(d.Context, project)

	if err != nil {
		return resp.Err("Error while fetching changelog entries [db query]", err)
	}

	entries := make([]types.ChangelogEntry, len(rows))
	for i, row := range rows {
		entries[i] = types.ChangelogEntry{
			Itag:             row.Itag,
			Project:          row.Project,
			Version:          row.Version,
			Added:            row.Added,
			Updated:          row.Updated,
			Fixed:            row.Fixed,
			Removed:          row.Removed,
			ExtraDescription: row.ExtraDescription,
			Prerelease:       row.Prerelease,
			CreatedBy:        row.CreatedBy,
			CreatedAt:        row.CreatedAt.Time,
		}
	}

	for i := range entries {
		entries[i].Author, err = dovewing.GetUser(d.Context, entries[i].CreatedBy, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error while getting user [dovewing]", err, zap.String("user_id", entries[i].CreatedBy))
		}
	}

	return uapi.HttpResponse{
		Json: types.ChangelogList{
			Entries: entries,
		},
	}
}
