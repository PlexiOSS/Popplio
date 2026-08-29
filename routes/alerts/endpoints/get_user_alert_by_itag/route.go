// Package get_user_alert_by_itag implements GET /users/{id}/alerts/{itag} —
// "Get User Alert By Itag".
//
// Gets a single user alert based on its `itag`. This returns an alertlist to
// aid with consistency.
package get_user_alert_by_itag

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Alert By Itag",
		Description: "Gets a single user alert based on its `itag`. This returns an alertlist to aid with consistency.",
		Resp:        types.AlertList{},
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "itag",
				Description: "The itag of the alert",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	itag := chi.URLParam(r, "itag")

	var itagUUID pgtype.UUID
	if err := itagUUID.Scan(itag); err != nil {
		return resp.Err("Invalid itag", err, zap.String("itag", itag), zap.String("userID", d.Auth.ID))
	}

	q := db.New(state.Pool)

	row, err := q.GetUserAlertByItag(d.Context, db.GetUserAlertByItagParams{
		UserID: d.Auth.ID,
		Itag:   itagUUID,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error querying for alert [collect]", err, zap.String("itag", itag), zap.String("userID", d.Auth.ID))
	}

	alert := types.Alert{
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

	unackedCount, err := q.CountUnackedUserAlerts(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Err("Error querying for unacked count", err, zap.String("itag", itag), zap.String("userID", d.Auth.ID))
	}

	return uapi.HttpResponse{
		Json: types.AlertList{
			Alerts: []types.Alert{
				alert,
			},
			UnackedCount: uint64(unackedCount),
		},
	}
}
