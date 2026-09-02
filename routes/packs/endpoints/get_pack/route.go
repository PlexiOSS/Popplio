// Copyright (C) 2026 NodeByte LTD

package get_pack

import (
	"errors"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Pack",
		Description: "Gets a pack on the list based on the URL.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The URL of the pack.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.BotPack{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	if id == "" {
		return uapi.DefaultResponse(http.StatusBadRequest)
	}

	row, err := db.New(state.Pool).GetPackByURL(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error querying packs table [collect]", err, zap.String("url", id))
	}

	pack := types.BotPack{
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

	err = assets.ResolveBotPack(d.Context, &pack)

	if err != nil {
		return resp.ErrDetail("Error resolving bot pack", err, zap.String("url", id))
	}

	return uapi.HttpResponse{
		Json: pack,
	}
}
