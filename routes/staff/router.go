// Package staff mounts the "Staff" group of API routes.
//
// Staff-only Omniplex endpoints. Only usable from staff panel using panelapi
// credentials
package staff

import (
	"popplio/routes/staff/endpoints/get_app_list"
	"popplio/routes/staff/endpoints/get_audit_events"
	"popplio/routes/staff/endpoints/get_auto_vote_overview"
	"popplio/routes/staff/endpoints/get_payments"
	"popplio/routes/staff/endpoints/get_public_team"
	"popplio/routes/staff/endpoints/get_shop_purchases"
	"popplio/routes/staff/endpoints/get_staff_permissions"
	"popplio/routes/staff/endpoints/manage_app"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const (
	tagName = "Staff"
)

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Staff-only Omniplex endpoints. Only usable from staff panel using panelapi credentials"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/staff/meta/permissions",
		OpId:    "get_staff_permissions",
		Method:  uapi.GET,
		Docs:    get_staff_permissions.Docs,
		Handler: get_staff_permissions.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/apps",
		OpId:    "get_app_list",
		Method:  uapi.GET,
		Docs:    get_app_list.Docs,
		Handler: get_app_list.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/audit-events",
		OpId:    "get_audit_events",
		Method:  uapi.GET,
		Docs:    get_audit_events.Docs,
		Handler: get_audit_events.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/auto-vote",
		OpId:    "get_auto_vote_overview",
		Method:  uapi.GET,
		Docs:    get_auto_vote_overview.Docs,
		Handler: get_auto_vote_overview.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/payments",
		OpId:    "get_payments",
		Method:  uapi.GET,
		Docs:    get_payments.Docs,
		Handler: get_payments.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/shop-purchases",
		OpId:    "get_shop_purchases",
		Method:  uapi.GET,
		Docs:    get_shop_purchases.Docs,
		Handler: get_shop_purchases.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/apps/{app_id}",
		OpId:    "manage_app",
		Method:  uapi.PATCH,
		Docs:    manage_app.Docs,
		Handler: manage_app.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/staff/team",
		OpId:    "get_public_team",
		Method:  uapi.GET,
		Docs:    get_public_team.Docs,
		Handler: get_public_team.Route,
	}.Route(r)
}
