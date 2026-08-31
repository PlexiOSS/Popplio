// Package servers mounts the "Servers" group of API routes.
//
// These API endpoints are related to servers on IBL
package servers

import (
	"net/http"

	"popplio/api"
	"popplio/perms"
	"popplio/routes/servers/endpoints/add_server"
	"popplio/routes/servers/endpoints/get_all_servers"
	"popplio/routes/servers/endpoints/get_flat_emojis"
	"popplio/routes/servers/endpoints/get_flat_stickers"
	"popplio/routes/servers/endpoints/get_random_servers"
	"popplio/routes/servers/endpoints/get_server"
	"popplio/routes/servers/endpoints/get_server_meta"
	"popplio/routes/servers/endpoints/get_server_seo"
	"popplio/routes/servers/endpoints/get_servers_emojis"
	"popplio/routes/servers/endpoints/get_servers_index"
	"popplio/routes/servers/endpoints/get_similar_servers"
	"popplio/routes/servers/endpoints/patch_server_settings"
	"popplio/routes/servers/endpoints/post_server_stats"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Servers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to servers on IBL"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/servers/@all",
		OpId:    "get_all_servers",
		Method:  uapi.GET,
		Docs:    get_all_servers.Docs,
		Handler: get_all_servers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@emojis",
		OpId:    "get_servers_emojis",
		Method:  uapi.GET,
		Docs:    get_servers_emojis.Docs,
		Handler: get_servers_emojis.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@emojis/flat",
		OpId:    "get_flat_emojis",
		Method:  uapi.GET,
		Docs:    get_flat_emojis.Docs,
		Handler: get_flat_emojis.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@stickers/flat",
		OpId:    "get_flat_stickers",
		Method:  uapi.GET,
		Docs:    get_flat_stickers.Docs,
		Handler: get_flat_stickers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/stats",
		OpId:    "post_server_stats",
		Method:  uapi.POST,
		Docs:    post_server_stats.Docs,
		Handler: post_server_stats.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeServer,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil, // No authorization is needed for this endpoint beyond defaults
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@random",
		OpId:    "get_random_servers",
		Method:  uapi.GET,
		Docs:    get_random_servers.Docs,
		Handler: get_random_servers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/@index",
		OpId:    "get_servers_index",
		Method:  uapi.GET,
		Docs:    get_servers_index.Docs,
		Handler: get_servers_index.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/meta",
		OpId:    "get_server_meta",
		Method:  uapi.GET,
		Docs:    get_server_meta.Docs,
		Handler: get_server_meta.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil,
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers",
		OpId:    "add_server",
		Method:  uapi.PUT,
		Docs:    add_server.Docs,
		Handler: add_server.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil,
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}",
		OpId:    "get_server",
		Method:  uapi.GET,
		Docs:    get_server.Docs,
		Handler: get_server.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/seo",
		OpId:    "get_server_seo",
		Method:  uapi.GET,
		Docs:    get_server_seo.Docs,
		Handler: get_server_seo.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/similar",
		OpId:    "get_similar_servers",
		Method:  uapi.GET,
		Docs:    get_similar_servers.Docs,
		Handler: get_similar_servers.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/settings",
		OpId:    "patch_server_settings",
		Method:  uapi.PATCH,
		Docs:    patch_server_settings.Docs,
		Handler: patch_server_settings.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perms.EntityEditServers),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return api.TargetTypeServer, chi.URLParam(r, "id")
				},
			},
		},
	}.Route(r)
}
