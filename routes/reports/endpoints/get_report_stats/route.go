// Package get_report_stats implements GET /reports/stats — "Get Report
// Stats".
//
// Returns anonymized counts of reports grouped by reason and status — no
// report IDs, no target identity, no reporter identity. A deliberate
// public surface (unlike the rest of the reports system, which has no
// public listing/review route at all) for moderation transparency.
package get_report_stats

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Report Stats",
		Description: "Returns anonymized counts of reports grouped by reason and status",
		Resp:        []types.ReportStatCount{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := state.Pool.Query(d.Context, "SELECT reason, status, COUNT(*) AS count FROM reports GROUP BY reason, status ORDER BY reason, status")

	if err != nil {
		return resp.Err("Error while querying report stats [db fetch]", err)
	}

	stats, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ReportStatCount])

	if errors.Is(err, pgx.ErrNoRows) {
		stats = []types.ReportStatCount{}
	} else if err != nil {
		return resp.Err("Error while querying report stats [collect]", err)
	}

	return uapi.HttpResponse{
		Json: stats,
	}
}
