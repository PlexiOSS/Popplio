// Package update_notification_prefs implements PATCH
// /users/{id}/notification-prefs — "Update User Notification Preferences".
//
// Partial update: only the categories present in the request body are
// changed, everything else keeps its existing (or default-enabled) state.
// Unknown category keys are rejected rather than silently ignored, so a
// typo'd category name doesn't look like it took effect when it didn't.
package update_notification_prefs

import (
	"net/http"
	"slices"

	"popplio/api/resp"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Update User Notification Preferences",
		Description: "Partially updates the calling user's notification preferences -- only the categories present in the request body are changed.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.NotificationPrefs{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var patch types.NotificationPrefs

	hresp, ok := uapi.MarshalReq(r, &patch)

	if !ok {
		return hresp
	}

	id := chi.URLParam(r, "id")

	for category, enabled := range patch {
		if !slices.Contains(types.AllAlertCategories, category) {
			return uapi.HttpResponse{
				Status: http.StatusBadRequest,
				Json:   types.ApiError{Message: "Unknown notification category: " + string(category)},
			}
		}

		_, err := state.Pool.Exec(
			d.Context,
			"INSERT INTO user_notification_prefs (user_id, category, enabled) VALUES ($1, $2, $3) ON CONFLICT (user_id, category) DO UPDATE SET enabled = $3",
			id, category, enabled,
		)

		if err != nil {
			return resp.Err("Failed to update notification preferences", err, zap.String("user_id", id), zap.String("category", string(category)))
		}
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
