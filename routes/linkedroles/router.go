// Copyright (C) 2026 NodeByte LTD

package linkedroles

import (
	"popplio/routes/linkedroles/endpoints/connect_linked_roles"
	"popplio/routes/linkedroles/endpoints/get_linked_roles"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Linked Roles"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "Discord linked roles backed by Omniplex badges"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/linked-roles",
		OpId:    "get_linked_roles",
		Method:  uapi.GET,
		Docs:    get_linked_roles.Docs,
		Handler: get_linked_roles.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/linked-roles/connect",
		OpId:    "connect_linked_roles",
		Method:  uapi.POST,
		Docs:    connect_linked_roles.Docs,
		Handler: connect_linked_roles.Route,
	}.Route(r)
}
