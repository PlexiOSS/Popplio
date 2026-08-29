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
	"popplio/db"
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

	rows, err := db.New(state.Pool).GetUserNotificationPrefs(d.Context, id)

	if err != nil {
		return resp.Err("Failed to get notification preferences", err, zap.String("user_id", id))
	}

	stored := map[types.AlertCategory]bool{}

	for _, row := range rows {
		stored[row.Category] = row.Enabled
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
