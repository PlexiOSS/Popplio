package get_team_permissions

import (
	"net/http"

	"popplio/perms"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team Permissions",
		Description: "Gets all permissions that a team can have",
		Resp:        types.PermissionResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	return uapi.HttpResponse{
		Json: types.PermissionResponse{
			Perms: types.PermissionDataFrom(perms.Entity.Definitions()),
		},
	}
}
