// Package changelogs mounts the "Changelog" group of API routes.
//
// These API endpoints are related to the curated, staff-authored release
// history for Popplio, Omniplex, and Keel.
package changelogs

import (
	"popplio/routes/changelogs/endpoints/get_changelog_list"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Changelog"

type Router struct{}

func (c Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to the curated, staff-authored release history for Popplio, Omniplex, and Keel."
}

func (c Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/changelogs/@all",
		OpId:    "get_changelog_list",
		Method:  uapi.GET,
		Docs:    get_changelog_list.Docs,
		Handler: get_changelog_list.Route,
	}.Route(r)
}
