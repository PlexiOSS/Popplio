// Package get_notification_prefs implements GET
// /users/{id}/notification-prefs — "Get User Notification Preferences".
//
// Returns every known alert category and whether the user wants
// notifications for it. A category with no stored row defaults to true --
// this is an opt-out model, so a user who's never touched their
// preferences sees everything, same as before preferences existed.
package get_notification_prefs

import (
	"net/http"

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
		Summary:     "Get User Notification Preferences",
		Description: "Returns every known alert category and whether the user wants notifications for it. A category with no stored row defaults to true.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.NotificationPrefs{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	rows, err := state.Pool.Query(d.Context, "SELECT category, enabled FROM user_notification_prefs WHERE user_id = $1", id)

	if err != nil {
		return resp.Err("Failed to get notification preferences", err, zap.String("user_id", id))
	}

	defer rows.Close()

	stored := map[types.AlertCategory]bool{}

	for rows.Next() {
		var category types.AlertCategory
		var enabled bool

		if err := rows.Scan(&category, &enabled); err != nil {
			return resp.Err("Failed to get notification preferences", err, zap.String("user_id", id))
		}

		stored[category] = enabled
	}

	if err := rows.Err(); err != nil {
		return resp.Err("Failed to get notification preferences", err, zap.String("user_id", id))
	}

	prefs := make(types.NotificationPrefs, len(types.AllAlertCategories))

	for _, category := range types.AllAlertCategories {
		if enabled, ok := stored[category]; ok {
			prefs[category] = enabled
		} else {
			prefs[category] = true
		}
	}

	return uapi.HttpResponse{
		Json: prefs,
	}
}
