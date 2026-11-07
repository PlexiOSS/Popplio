// Package get_apps_meta implements GET /apps/meta — "Get Apps Meta".
//
// Gets the current applications metadata. Returns a `AppMeta` object. See
// schema for more info.
package get_apps_meta

import (
	"net/http"

	"popplio/api/resp"
	"popplio/apps"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Apps Meta",
		Description: "Gets the current applications metadata. Returns a ``AppMeta`` object. See schema for more info.",
		Resp:        types.AppMeta{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	positions, err := apps.List(d.Context)

	if err != nil {
		return resp.Err("Failed to load application positions", err)
	}

	return uapi.HttpResponse{
		Json: types.AppMeta{
			Positions: positions,
			Stable:    apps.Stable,
		},
	}
}
