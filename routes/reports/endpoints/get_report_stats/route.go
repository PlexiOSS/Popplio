// Package get_report_stats implements GET /reports/stats — "Get Report
// Stats".
//
// Returns anonymized counts of reports grouped by reason and status — no
// report IDs, no target identity, no reporter identity. A deliberate
// public surface (unlike the rest of the reports system, which has no
// public listing/review route at all) for moderation transparency.
package get_report_stats

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

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
	rows, err := db.New(state.Pool).GetReportStats(d.Context)

	if err != nil {
		return resp.Err("Error while querying report stats [db fetch]", err)
	}

	stats := make([]types.ReportStatCount, len(rows))
	for i, row := range rows {
		stats[i] = types.ReportStatCount{
			Reason: row.Reason,
			Status: row.Status,
			Count:  row.Count,
		}
	}

	return uapi.HttpResponse{
		Json: stats,
	}
}
