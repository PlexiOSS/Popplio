// Package get_user_notifications implements GET /users/{id}/notifications —
// "Get User Notifications".
//
// Gets a users notifications
package get_user_notifications

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
	ua "github.com/mileusna/useragent"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Notifications",
		Description: "Gets a users notifications",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.NotifGetList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var id = chi.URLParam(r, "id")

	rows, err := db.New(state.Pool).GetUserNotifications(d.Context, id)

	if err != nil {
		return resp.Err("Failed to get user notifications", err, zap.String("user_id", id))
	}

	notifications := make([]types.NotifGet, len(rows))
	for i, row := range rows {
		notifications[i] = types.NotifGet{
			Endpoint:  row.Endpoint,
			NotifID:   row.NotifID,
			CreatedAt: row.CreatedAt.Time,
			UA:        row.Ua,
		}
	}

	for i := range notifications {
		uaD := ua.Parse(notifications[i].UA)

		notifications[i].BrowserInfo = types.NotifBrowserInfo{
			OS:         uaD.OS,
			Browser:    uaD.Name,
			BrowserVer: uaD.Version,
			Mobile:     uaD.Mobile,
		}
	}

	sublist := types.NotifGetList{
		Notifications: notifications,
	}

	return uapi.HttpResponse{
		Json: sublist,
	}
}
