// Package get_featured_user_alerts implements GET
// /users/{id}/alerts/@featured — "Get Featured User Alerts".
//
// Gets the featured user alerts of the user.
package get_featured_user_alerts

import (
	"net/http"
	"strconv"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func toAlert(row db.GetUserAlertsByAckedRow) types.Alert {
	return types.Alert{
		ITag:      row.Itag,
		URL:       row.Url,
		Message:   row.Message,
		Type:      row.Type,
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
		Acked:     row.Acked,
		AlertData: row.AlertData,
		Icon:      row.Icon.String,
		Priority:  row.Priority,
		Category:  row.Category,
	}
}

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Featured User Alerts",
		Description: "Gets the featured user alerts of the user.",
		Resp:        types.FeaturedUserAlerts{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "acked_count",
				Description: "The number of alerts to return that have been acknowledged.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "unacked_count",
				Description: "The number of alerts to return that have not been acknowledged.",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	ackedResCount, err := strconv.Atoi(r.URL.Query().Get("acked_count"))

	if err != nil {
		return resp.BadRequest("acked_count must be an integer")
	}

	if ackedResCount > 20 {
		return resp.BadRequest("acked_count must be less than or equal to 20")
	}

	unackedResCount, err := strconv.Atoi(r.URL.Query().Get("unacked_count"))

	if err != nil {
		return resp.BadRequest("unacked_count must be an integer")
	}

	if unackedResCount > 20 {
		return resp.BadRequest("unacked_count must be less than or equal to 20")
	}

	q := db.New(state.Pool)

	ackedRows, err := q.GetUserAlertsByAcked(d.Context, db.GetUserAlertsByAckedParams{
		UserID: d.Auth.ID,
		Acked:  true,
		Limit:  int32(ackedResCount),
	})

	if err != nil {
		return resp.Err("Error getting acked user alerts [query]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	ackedAlerts := make([]types.Alert, len(ackedRows))
	for i, row := range ackedRows {
		ackedAlerts[i] = toAlert(row)
	}

	unackedRows, err := q.GetUserAlertsByAcked(d.Context, db.GetUserAlertsByAckedParams{
		UserID: d.Auth.ID,
		Acked:  false,
		Limit:  int32(unackedResCount),
	})

	if err != nil {
		return resp.Err("Error getting unacked user alerts [query]", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	unackedAlerts := make([]types.Alert, len(unackedRows))
	for i, row := range unackedRows {
		unackedAlerts[i] = toAlert(row)
	}

	unackedCount, err := q.CountUnackedUserAlerts(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error getting unacked user alerts count", err, zap.String("userID", d.Auth.ID), zap.Int("ackedResCount", ackedResCount), zap.Int("unackedResCount", unackedResCount))
	}

	return uapi.HttpResponse{
		Json: types.FeaturedUserAlerts{
			UnackedCount: uint64(unackedCount),
			Unacked:      unackedAlerts,
			Acked:        ackedAlerts,
		},
	}
}
