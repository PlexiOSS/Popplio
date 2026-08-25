// Package reports mounts the "Reports" group of API routes.
//
// These endpoints let a user file a report against any votable entity, and
// let anyone read back anonymized aggregate counts (reason/status only, no
// IDs). Individual reports themselves have no public listing/review route
// — those are only ever read back through the Arcadia staff panel (see
// arcadia/panel/ops_reports.go).
package reports

import (
	"popplio/api"
	"popplio/routes/reports/endpoints/create_report"
	"popplio/routes/reports/endpoints/get_report_stats"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Reports"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints let a user file a report against any votable entity"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/reports/stats",
		OpId:    "get_report_stats",
		Method:  uapi.GET,
		Docs:    get_report_stats.Docs,
		Handler: get_report_stats.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/{target_type}/{target_id}/reports",
		OpId:    "create_report",
		Method:  uapi.PUT,
		Docs:    create_report.Docs,
		Handler: create_report.Route,
		Auth: []uapi.AuthType{
			{
				Type:   api.TargetTypeUser,
				URLVar: "uid",
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
