// Package themes mounts the "Themes" group of API routes.
//
// These API endpoints are related to user-submitted Discord profile themes
// (a name plus two hex colors) -- simply-owned entities, no team
// permissions, closest in shape to pack emojis/stickers.
package themes

import (
	"popplio/api"
	"popplio/routes/themes/endpoints/add_theme"
	"popplio/routes/themes/endpoints/delete_theme"
	"popplio/routes/themes/endpoints/get_all_themes"
	"popplio/routes/themes/endpoints/get_theme"

	"github.com/go-chi/chi/v5"

	"github.com/PlexiOSS/Keel/uapi"
)

const tagName = "Themes"

type Router struct{}

func (b Router) Tag() (string, string) {
	return tagName, "These API endpoints are related to user-submitted Discord profile themes"
}

func (b Router) Routes(r *chi.Mux) {
	uapi.Route{
		Pattern: "/themes/@all",
		OpId:    "get_all_themes",
		Method:  uapi.GET,
		Docs:    get_all_themes.Docs,
		Handler: get_all_themes.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/themes/{id}",
		OpId:    "get_theme",
		Method:  uapi.GET,
		Docs:    get_theme.Docs,
		Handler: get_theme.Route,
	}.Route(r)

	uapi.Route{
		Pattern: "/users/{id}/themes",
		OpId:    "add_theme",
		Method:  uapi.PUT,
		Docs:    add_theme.Docs,
		Handler: add_theme.Route,
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
		Pattern: "/users/{uid}/themes/{id}",
		OpId:    "delete_theme",
		Method:  uapi.DELETE,
		Docs:    delete_theme.Docs,
		Handler: delete_theme.Route,
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
