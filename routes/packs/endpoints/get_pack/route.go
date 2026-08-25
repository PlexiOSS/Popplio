// Package get_pack implements GET /packs/{id} — "Get Pack".
//
// Gets a pack on the list based on the URL.
package get_pack

import (
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/routes/packs/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var (
	packColArr = dbutil.GetCols(types.BotPack{})
	packCols   = strings.Join(packColArr, ",")
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

	row, err := state.Pool.Query(d.Context, "SELECT "+packCols+" FROM packs WHERE url = $1", id)

	if err != nil {
		return resp.Err("Error querying packs table [db fetch]", err, zap.String("url", id))
	}

	pack, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.BotPack])

	if err == pgx.ErrNoRows {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error querying packs table [collect]", err, zap.String("url", id))
	}

	err = assets.ResolveBotPack(d.Context, &pack)

	if err != nil {
		return resp.ErrDetail("Error resolving bot pack", err, zap.String("url", id))
	}

	return uapi.HttpResponse{
		Json: pack,
	}
}
