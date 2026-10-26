// Package get_main_server_status implements GET /users/{id}/main-server —
// "Get Main Server Status".
//
// Whether the signed-in user is a member of the Omniplex Discord server.
package get_main_server_status

import (
	"net/http"

	"popplio/api/resp"
	"popplio/mainserver"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Main Server Status",
		Description: "Returns whether the user is a member of the Omniplex Discord server. Only the user themselves can check.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.MainServerStatus{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	member, err := mainserver.IsMember(chi.URLParam(r, "id"))

	if err != nil {
		return resp.Err("Failed to check server membership", err)
	}

	return uapi.HttpResponse{Json: types.MainServerStatus{Member: member}}
}
