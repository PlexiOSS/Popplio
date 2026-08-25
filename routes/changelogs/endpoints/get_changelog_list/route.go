// Package get_changelog_list implements GET /changelogs/@all — "Get
// Changelog List".
//
// Gets published changelog entries for Popplio and/or Omniplex, newest
// first.
package get_changelog_list

import (
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	changelogColsArr = dbutil.GetCols(types.ChangelogEntry{})
	changelogCols    = strings.Join(changelogColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Changelog List",
		Description: "Gets published changelog entries for Popplio and/or Omniplex, newest first.",
		Params: []docs.Parameter{
			{
				Name:        "project",
				Description: "Filter to one project: 'popplio' or 'omniplex'. Omit for both.",
				Required:    false,
				In:          "query",
			},
		},
		Resp: types.ChangelogList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var project *string

	if v := r.URL.Query().Get("project"); v != "" {
		project = &v
	}

	rows, err := state.Pool.Query(d.Context,
		"SELECT "+changelogCols+" FROM changelogs WHERE published = true AND ($1::text IS NULL OR project = $1) ORDER BY created_at DESC",
		project)

	if err != nil {
		return resp.Err("Error while fetching changelog entries [db query]", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ChangelogEntry])

	if err != nil {
		return resp.Err("Error while fetching changelog entries [collect]", err)
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
