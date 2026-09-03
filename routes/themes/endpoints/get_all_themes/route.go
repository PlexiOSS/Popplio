// Copyright (C) 2026 NodeByte LTD

package get_all_themes

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	"github.com/PlexiOSS/Keel/dovewing"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 24

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Themes",
		Description: "Gets every theme, paginated, newest first",
		Resp:        types.PagedResult[[]types.Theme]{},
		RespName:    "PagedResultTheme",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "owner",
				Description: "Filter to themes submitted by this user ID. Omit to return every submitter's.",
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
	offset := (pageNum - 1) * perPage
	owner := r.URL.Query().Get("owner")

	q := db.New(state.Pool)

	rows, err := q.GetAllThemesPaged(d.Context, db.GetAllThemesPagedParams{
		Limit:  int32(limit),
		Offset: int32(offset),
		Owner:  owner,
	})

	if err != nil {
		return resp.Err("Error while querying themes [db fetch]", err)
	}

	themes := make([]types.Theme, len(rows))

	for i, row := range rows {
		owner, err := dovewing.GetUser(d.Context, row.Owner, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error querying dovewing for owner user", err)
		}

		themes[i] = types.Theme{
			ID:             row.ID,
			Name:           row.Name,
			PrimaryColor:   row.PrimaryColor,
			SecondaryColor: row.SecondaryColor,
			Tags:           row.Tags,
			CreatedAt:      row.CreatedAt.Time,
			ResolvedOwner:  owner,
		}
	}

	count, err := q.CountThemes(d.Context, owner)

	if err != nil {
		return resp.Err("Error while querying themes [db count]", err)
	}

	return uapi.HttpResponse{
		Json: types.PagedResult[[]types.Theme]{
			Count:   uint64(count),
			PerPage: perPage,
			Results: themes,
		},
	}
}
