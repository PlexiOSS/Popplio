// Package servers mounts the "Servers" group of API routes.
//
// These API endpoints are related to servers on Omniplex
package servers

import (
	"net/http"

	"popplio/views"
	"popplio/api"
	"popplio/perms"
	"popplio/routes/entityassets/put_entity_asset"
	"popplio/routes/pagestyle"
	"popplio/routes/servers/endpoints/add_server"
	"popplio/routes/servers/endpoints/create_server_changelog"
	"popplio/routes/servers/endpoints/delete_server"
	"popplio/routes/servers/endpoints/delete_server_changelog"
	"popplio/routes/servers/endpoints/get_all_servers"
	"popplio/routes/servers/endpoints/get_flat_emojis"
	"popplio/routes/servers/endpoints/get_flat_stickers"
	"popplio/routes/servers/endpoints/get_random_servers"
	"popplio/routes/servers/endpoints/get_server"
	"popplio/routes/servers/endpoints/get_server_analytics"
	"popplio/routes/servers/endpoints/get_server_bumps"
	"popplio/routes/servers/endpoints/get_server_changelogs"
	"popplio/routes/servers/endpoints/get_server_changelogs_feed"
	"popplio/routes/servers/endpoints/get_server_meta"
	"popplio/routes/servers/endpoints/get_server_seo"
	"popplio/routes/servers/endpoints/get_server_vote_channel"
	"popplio/routes/servers/endpoints/get_servers_emojis"
	"popplio/routes/servers/endpoints/get_servers_index"
	"popplio/routes/servers/endpoints/get_similar_servers"
	"popplio/routes/servers/endpoints/patch_server_settings"
	"popplio/routes/servers/endpoints/post_server_stats"
	"popplio/routes/servers/endpoints/put_server_vote_channel"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Servers"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to servers on Omniplex"
}

func (b Router) Routes(r *chi.Mux) {
	put_entity_asset.Mount(r, api.TargetTypeServer, "/servers", "id", perms.EntityEditServers)
	pagestyle.Mount(r, api.TargetTypeServer, "/servers", perms.EntityEditServers)

	uapi.Route{
		Pattern: "/servers/{id}/views",
		OpId:    "record_server_view",
		Method:  uapi.POST,
		Docs:    views.Docs(views.EventView, "server"),
		Handler: views.Route(views.EventView, "server"),
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/invite-clicks",
		OpId:    "record_server_invite_click",
		Method:  uapi.POST,
		Docs:    views.Docs(views.EventInvite, "server"),
		Handler: views.Route(views.EventInvite, "server"),
	}.Route(r)

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
		OpId:    "delete_server",
		Method:  uapi.DELETE,
		Docs:    delete_server.Docs,
		Handler: delete_server.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeTeam,
			},
		},
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: api.PermissionCheck{
				NeededPermission: api.Needs(perms.EntityDeleteServers),
				GetTarget: func(d uapi.Route, r *http.Request, authData uapi.AuthData) (string, string) {
					return api.TargetTypeServer, chi.URLParam(r, "id")
				},
			},
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}",
		OpId:    "get_server",
		Method:  uapi.GET,
		Docs:    get_server.Docs,
		Handler: get_server.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
			{
				Type: api.TargetTypeServer,
			},
		},
		AuthOptional: true,
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil,
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/bumps",
		OpId:    "get_server_bumps",
		Method:  uapi.GET,
		Docs:    get_server_bumps.Docs,
		Handler: get_server_bumps.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/analytics",
		OpId:    "get_server_analytics",
		Method:  uapi.GET,
		Docs:    get_server_analytics.Docs,
		Handler: get_server_analytics.Route,
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

	uapi.Route{
		Pattern: "/servers/@changelogs",
		OpId:    "get_server_changelogs_feed",
		Method:  uapi.GET,
		Docs:    get_server_changelogs_feed.Docs,
		Handler: get_server_changelogs_feed.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/changelogs",
		OpId:    "get_server_changelogs",
		Method:  uapi.GET,
		Docs:    get_server_changelogs.Docs,
		Handler: get_server_changelogs.Route,
		Auth: []uapi.AuthType{
			{
				Type: api.TargetTypeUser,
			},
		},
		AuthOptional: true,
		ExtData: map[string]any{
			api.PERMISSION_CHECK_KEY: nil,
		},
	}.Route(r)

	uapi.Route{
		Pattern: "/servers/{id}/changelogs",
		OpId:    "create_server_changelog",
		Method:  uapi.POST,
		Docs:    create_server_changelog.Docs,
		Handler: create_server_changelog.Route,
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

	uapi.Route{
		Pattern: "/servers/{id}/changelogs/{changelog_id}",
		OpId:    "delete_server_changelog",
		Method:  uapi.DELETE,
		Docs:    delete_server_changelog.Docs,
		Handler: delete_server_changelog.Route,
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

	uapi.Route{
		Pattern: "/servers/{id}/vote-channel",
		OpId:    "get_server_vote_channel",
		Method:  uapi.GET,
		Docs:    get_server_vote_channel.Docs,
		Handler: get_server_vote_channel.Route,
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

	uapi.Route{
		Pattern: "/servers/{id}/vote-channel",
		OpId:    "put_server_vote_channel",
		Method:  uapi.PUT,
		Docs:    put_server_vote_channel.Docs,
		Handler: put_server_vote_channel.Route,
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
