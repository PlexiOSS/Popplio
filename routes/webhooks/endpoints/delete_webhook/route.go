// Package delete_webhook implements DELETE
// /{target_type}/{target_id}/webhooks/{webhook_id} — "Delete Webhook".
//
// Updates an existing webhook on an entity. Returns 204 on success.
package delete_webhook

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

const MaximumWebhookCount = 5

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Webhook",
		Description: "Updates an existing webhook on an entity. Returns 204 on success. **Requires Edit Webhooks permission**",
		Resp:        types.ApiError{},
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
				Name:        "webhook_id",
				Description: "The ID of the webhook to delete",
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
	webhookId := chi.URLParam(r, "webhook_id")

	if targetId == "" || targetType == "" || webhookId == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	switch targetType {
	case "bot":
	case "server":
	case "team":
	default:
		return resp.Status(http.StatusNotImplemented, "Creating webhooks for this target type is not yet supported")
	}

	var webhookUUID pgtype.UUID
	if err := webhookUUID.Scan(webhookId); err != nil {
		return resp.NotFound("Webhook not found")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error while starting transaction", err, zap.String("userID", d.Auth.ID))
	}

	q := db.New(tx)

	count, err := q.CountWebhookByID(d.Context, db.CountWebhookByIDParams{
		TargetID:   targetId,
		TargetType: targetType,
		ID:         webhookUUID,
	})

	if err != nil {
		return resp.Err("Error while checking webhook", err, zap.String("userID", d.Auth.ID))
	}

	if count == 0 {
		return resp.NotFound("Webhook not found")
	}

	err = q.DeleteWebhook(d.Context, db.DeleteWebhookParams{
		TargetID:   targetId,
		TargetType: targetType,
		ID:         webhookUUID,
	})

	if err != nil {
		return resp.Err("Error while inserting webhook", err, zap.String("userID", d.Auth.ID))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error while committing transaction", err, zap.String("userID", d.Auth.ID))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
