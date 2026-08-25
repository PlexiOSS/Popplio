// Package badges mounts the "Badges" group of API routes.
//
// Public and read-only — badges are assigned by staff through Arcadia's
// panel/RPC layer, not this API.
package badges

import (
	"popplio/routes/badges/endpoints/get_entity_badges"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Badges"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to the badge system on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/{target_type}/{target_id}/badges",
		OpId:    "get_entity_badges",
		Method:  uapi.GET,
		Docs:    get_entity_badges.Docs,
		Handler: get_entity_badges.Route,
	}.Route(r)
}
