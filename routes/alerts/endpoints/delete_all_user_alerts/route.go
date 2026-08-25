// Package delete_all_user_alerts implements DELETE /users/{id}/alerts —
// "Delete All User Alerts".
//
// Deletes all user alerts. Returns 204 on success
package delete_all_user_alerts

import (
	"net/http"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete All User Alerts",
		Description: "Deletes all user alerts. Returns 204 on success",
		Resp:        types.ApiError{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	_, err := state.Pool.Exec(d.Context, "DELETE FROM alerts WHERE user_id = $1", d.Auth.ID)

	if err != nil {
		return resp.Err("Failed to delete alerts", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
