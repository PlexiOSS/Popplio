// Package delete_user_notifications implements DELETE
// /users/{id}/notification — "Delete User Notifications".
//
// Deletes a users notification. Returns 204 on success
package delete_user_notifications

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
		Summary:     "Delete User Notifications",
		Description: "Deletes a users notification. Returns 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "notif_id",
				Description: "Notification ID",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")
	notifId := r.URL.Query().Get("notif_id")

	// Check for notif_id
	if notifId == "" {
		return resp.BadRequest("`notif_id` is required in query params and must be set to the notification ID to delete")
	}

	q := db.New(state.Pool)

	// Check count of deleted rows
	count, err := q.CountUserNotification(d.Context, db.CountUserNotificationParams{
		UserID:  id,
		NotifID: notifId,
	})

	if err != nil {
		return resp.ErrDetail("Error while checking user notification count", err, zap.String("userID", id), zap.String("notifID", r.URL.Query().Get("notif_id")))
	}

	if count == 0 {
		return resp.NotFound("Notification not found")
	}

	err = q.DeleteUserNotification(d.Context, db.DeleteUserNotificationParams{
		UserID:  id,
		NotifID: notifId,
	})

	if err != nil {
		return resp.ErrDetail("Error while deleting user notification", err, zap.String("userID", id), zap.String("notifID", r.URL.Query().Get("notif_id")))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
