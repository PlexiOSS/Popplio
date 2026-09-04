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

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 12

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All Packs",
		Description: "Gets all packs on the list. This endpoint is paginated.",
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

	packType := r.URL.Query().Get("pack_type")

	q := db.New(state.Pool)
	var packs []types.BotPack
	var count int64

	if packType != "" {
		if packType != types.PackTypeBot && packType != types.PackTypeServer && packType != types.PackTypeEmoji && packType != types.PackTypeSticker && packType != types.PackTypeSound {
			return resp.BadRequest("pack_type must be one of bot, server, emoji, sticker, or sound")
		}

		rows, err := q.GetAllPacksByType(d.Context, db.GetAllPacksByTypeParams{
			PackType: packType,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})

		if err != nil {
			return resp.Err("Error while querying packs [db fetch]", err)
		}

		packs = make([]types.BotPack, len(rows))
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

		count, err = q.CountPacksByType(d.Context, packType)

		if err != nil {
			return resp.Err("Error while querying packs [db count]", err)
		}
	} else {
		rows, err := q.GetAllPacks(d.Context, db.GetAllPacksParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})

		if err != nil {
			return resp.Err("Error while querying packs [db fetch]", err)
		}

		packs = make([]types.BotPack, len(rows))
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

		count, err = q.CountPacks(d.Context)

		if err != nil {
			return resp.Err("Error while querying packs [db count]", err)
		}
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
