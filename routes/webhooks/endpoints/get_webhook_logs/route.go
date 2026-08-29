// Package get_webhook_logs implements GET
// /{target_type}/{target_id}/webhooks/logs — "Get Webhook Logs".
//
// Gets webhook logs of a specific entity. Paginated to 10 at a time.
package get_webhook_logs

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 10

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhook Logs",
		Description: "Gets webhook logs of a specific entity. Paginated to 10 at a time. **Requires authentication**",
		Resp:        types.PagedResult[[]types.WebhookLogEntry]{},
		RespName:    "PagedResultWebhookLogEntry",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
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

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	targetId := chi.URLParam(r, "target_id")

	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	q := db.New(state.Pool)

	// Fetch the logs
	rows, err := q.GetWebhookLogsPage(d.Context, db.GetWebhookLogsPageParams{
		TargetID:   targetId,
		TargetType: targetType,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})

	if err != nil {
		return resp.Err("Error while querying webhook logs [db fetch]", err, zap.String("userID", d.Auth.ID))
	}

	webhooks := make([]types.WebhookLogEntry, len(rows))
	for i, row := range rows {
		webhooks[i] = types.WebhookLogEntry{
			ID:              row.ID,
			WebhookID:       row.WebhookID,
			TargetID:        row.TargetID,
			TargetType:      row.TargetType,
			UserID:          row.UserID,
			URL:             row.Url,
			Data:            row.Data,
			Response:        row.Response,
			CreatedAt:       row.CreatedAt.Time,
			State:           row.State,
			Tries:           int(row.Tries),
			LastTry:         row.LastTry.Time,
			BadIntent:       row.BadIntent,
			StatusCode:      int(row.StatusCode),
			RequestHeaders:  row.RequestHeaders,
			ResponseHeaders: row.ResponseHeaders,
		}
	}

	for i, webhook := range webhooks {
		webhooks[i].User, err = dovewing.GetUser(d.Context, webhook.UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Error while querying webhook logs [dovewing]", err, zap.String("userID", d.Auth.ID))
		}
	}

	count, err := q.CountWebhookLogs(d.Context, db.CountWebhookLogsParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return resp.Err("Error while querying webhook logs [db count]", err, zap.String("userID", d.Auth.ID))
	}

	data := types.PagedResult[[]types.WebhookLogEntry]{
		Count:   uint64(count),
		Results: webhooks,
		PerPage: perPage,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
