// Package servertemplates mounts the "Server Templates" group of API
// routes.
//
// These API endpoints are related to user-submitted Discord server
// template listings
package servertemplates

import (
	"popplio/api"
	"popplio/routes/servertemplates/endpoints/add_server_template"
	"popplio/routes/servertemplates/endpoints/delete_server_template"
	"popplio/routes/servertemplates/endpoints/get_all_server_templates"
	"popplio/routes/servertemplates/endpoints/get_server_template"
	"popplio/routes/servertemplates/endpoints/get_template_reaction"
	"popplio/routes/servertemplates/endpoints/patch_server_template"
	"popplio/routes/servertemplates/endpoints/set_template_reaction"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Server Templates"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to user-submitted Discord server template listings"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/server-templates/@all",
		OpId:    "get_all_server_templates",
		Method:  uapi.GET,
		Docs:    get_all_server_templates.Docs,
		Handler: get_all_server_templates.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/server-templates/{id}",
		OpId:    "get_server_template",
		Method:  uapi.GET,
		Docs:    get_server_template.Docs,
		Handler: get_server_template.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/server-templates",
		OpId:    "add_server_template",
		Method:  uapi.PUT,
		Docs:    add_server_template.Docs,
		Handler: add_server_template.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "id",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/server-templates/{id}",
		OpId:    "delete_server_template",
		Method:  uapi.DELETE,
		Docs:    delete_server_template.Docs,
		Handler: delete_server_template.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/server-templates/{id}",
		OpId:    "patch_server_template",
		Method:  uapi.PATCH,
		Docs:    patch_server_template.Docs,
		Handler: patch_server_template.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/server-templates/{id}/reaction",
		OpId:    "get_template_reaction",
		Method:  uapi.GET,
		Docs:    get_template_reaction.Docs,
		Handler: get_template_reaction.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{uid}/server-templates/{id}/reaction",
		OpId:    "set_template_reaction",
		Method:  uapi.PUT,
		Docs:    set_template_reaction.Docs,
		Handler: set_template_reaction.Route,
		Auth: []uapi.AuthType{
			{
				URLVar: "uid",
				Type:   api.TargetTypeUser,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)
}
