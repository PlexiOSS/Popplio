// Package get_random_bots implements GET /bots/@random — "Get Random Bots".
//
// Returns a list of bots from the database in random order
package get_random_bots

import (
	"net/http"
	"strings"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/api/resp"
	"popplio/routes/bots/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

var (
	indexBotColsArr = dbutil.GetCols(types.IndexBot{})
	indexBotCols    = strings.Join(indexBotColsArr, ",")
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Random Bots",
		Description: "Returns a list of bots from the database in random order",
		Resp: types.RandomBots{
			Bots: []types.IndexBot{},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT "+indexBotCols+" FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY RANDOM() LIMIT 6")

	if err != nil {
		return resp.Err("Error while getting random bots [db fetch]", err)
	}

	bots, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.IndexBot])

	if err != nil {
		return resp.Err("Error while getting random bots [collect]", err)
	}

	// Resolve all bots concurrently, since each bot's resolution is independent
	if err := assets.ResolveIndexBots(d.Context, bots); err != nil {
		return resp.ErrBody("Error resolving indexbot", "An error occurred while resolving index bot.", err)
	}

	return uapi.HttpResponse{
		Json: types.RandomBots{
			Bots: bots,
		},
	}
}
