// Package get_user_alerts implements GET /users/{id}/alerts — "Get User
// Alerts".
//
// Gets a users alerts.
package get_user_alerts

import (
	"net/http"

	"popplio/api/resp"
	"popplio/pagination"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Alerts",
		Description: "Gets a users alerts.\n\nAll alerts are also sent via push notifications if the user has subscribed to them.",
		Resp:        types.PagedResult[types.AlertList]{},
		RespName:    "PagedResultAlertList",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "page",
				Description: "The page number",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

const perPage = 20

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Page must be an integer")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	q := db.New(state.Pool)

	rows, err := q.GetUserAlertsPage(d.Context, db.GetUserAlertsPageParams{
		UserID: d.Auth.ID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})

	if err != nil {
		return resp.Err("Error getting alerts [db]", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	alerts := make([]types.Alert, len(rows))
	for i, row := range rows {
		alerts[i] = types.Alert{
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

	count, err := q.CountUserAlerts(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error getting total alert count", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	unackedCount, err := q.CountUnackedUserAlerts(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error getting total unacked alert count", err, zap.String("userID", d.Auth.ID), zap.Int("limit", limit), zap.Uint64("offset", offset))
	}

	data := types.PagedResult[types.AlertList]{
		Count: uint64(count),
		Results: types.AlertList{
			UnackedCount: uint64(unackedCount),
			Alerts:       alerts,
		},
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
