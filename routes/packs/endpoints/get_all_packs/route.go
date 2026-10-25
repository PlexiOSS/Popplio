// Copyright (C) 2026 NodeByte LTD

package get_all_packs

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"

	"popplio/routes/packs/assets"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Packs",
		Description: "Gets packs on the list, filtered and sorted. Packs with a description under 20 characters are left out unless all=true. This endpoint is paginated.",
		Resp:        types.PagedResult[[]types.BotPack]{},
		RespName:    "PagedResultIndexBotPack",
		Params: []docs.Parameter{
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "pack_type",
				Description: "Filter to only packs of this type (bot, server, emoji, sticker, or sound). Omit to return every type.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "sort",
				Description: "new (default, newest first) or votes (net votes, then most bots and servers, then newest).",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "all",
				Description: "true to include packs with a description under 20 characters.",
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

	var packType pgtype.Text

	if raw := query.Get("pack_type"); raw != "" {
		if raw != types.PackTypeBot && raw != types.PackTypeServer && raw != types.PackTypeEmoji && raw != types.PackTypeSticker && raw != types.PackTypeSound {
			return resp.BadRequest("pack_type must be one of bot, server, emoji, sticker, or sound")
		}

		packType = pgtype.Text{String: raw, Valid: true}
	}

	sort := query.Get("sort")

	if sort == "" {
		sort = "new"
	}

	if sort != "new" && sort != "votes" {
		return resp.BadRequest("sort must be new or votes")
	}

	includeAll := query.Get("all") == "true"

	q := db.New(state.Pool)

	rows, err := q.BrowsePacks(d.Context, db.BrowsePacksParams{
		PackType:   packType,
		IncludeAll: includeAll,
		Sort:       sort,
		Lim:        perPage,
		Off:        int32((pageNum - 1) * perPage),
	})

	if err != nil {
		return resp.Err("Error while querying packs [db fetch]", err)
	}

	packs := make([]types.BotPack, len(rows))

	for i, row := range rows {
		packs[i] = types.BotPack{
			Owner:      row.Owner,
			Name:       row.Name,
			Short:      row.Short,
			Tags:       row.Tags,
			URL:        row.Url,
			CreatedAt:  row.CreatedAt.Time,
			PackType:   row.PackType,
			Bots:       row.Bots,
			Servers:    row.Servers,
			VoteBanned: row.VoteBanned,
		}
	}

	count, err := q.CountBrowsePacks(d.Context, db.CountBrowsePacksParams{
		PackType:   packType,
		IncludeAll: includeAll,
	})

	if err != nil {
		return resp.Err("Error while querying packs [db count]", err)
	}

	for i := range packs {
		err = assets.ResolveBotPack(d.Context, &packs[i])

		if err != nil {
			return resp.ErrDetail("Error resolving bot pack", err, zap.String("url", packs[i].URL))
		}
	}

	data := types.PagedResult[[]types.BotPack]{
		Count:   uint64(count),
		PerPage: perPage,
		Results: packs,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
