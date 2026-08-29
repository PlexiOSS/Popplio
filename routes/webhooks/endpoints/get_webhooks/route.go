// Package get_webhooks implements GET /{target_type}/{target_id}/webhooks —
// "Get Webhooks".
//
// Gets a list of webhooks of a specific entity (excluding the secret due to
// security concerns).
package get_webhooks

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Webhooks",
		Description: "Gets a list of webhooks of a specific entity (excluding the secret due to security concerns). **Requires the Get Webhooks permission**",
		Resp:        []types.Webhook{},
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The entity type to return webhooks for.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID to return webhooks for",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	rows, err := db.New(state.Pool).GetWebhooks(d.Context, db.GetWebhooksParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		return resp.Err("Error while querying webhooks [collect]", err, zap.String("userID", d.Auth.ID))
	}

	webhook := make([]types.Webhook, len(rows))
	for i, row := range rows {
		webhook[i] = types.Webhook{
			ID:             row.ID,
			Name:           row.Name,
			TargetID:       row.TargetID,
			TargetType:     row.TargetType,
			Url:            row.Url,
			Broken:         row.Broken,
			FailedRequests: int(row.FailedRequests),
			SimpleAuth:     row.SimpleAuth,
			HmacAuth:       row.HmacAuth,
			EventWhitelist: row.EventWhitelist,
			CreatedAt:      row.CreatedAt.Time,
		}
	}

	return uapi.HttpResponse{
		Json: webhook,
	}
}
