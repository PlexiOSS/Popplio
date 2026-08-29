package remove_review

import (
	"net/http"

	"popplio/api"
	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/reviews/assets"
	"popplio/state"
	"popplio/types"
	"popplio/validators"
	"popplio/webhooks/core/drivers"
	"popplio/webhooks/events"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Review",
		Description: "Deletes a review by review ID. The user must be the author of this review. This will automatically trigger a garbage collection task and returns 204 on success",
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
				Name:        "review_id",
				Description: "The review ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))
	rid := chi.URLParam(r, "review_id")

	q := db.New(state.Pool)

	var ridUUID pgtype.UUID
	if err := ridUUID.Scan(rid); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	review, err := q.GetReviewForModify(d.Context, db.GetReviewForModifyParams{
		ID:         ridUUID,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to query review [db queryrow]", zap.Error(err), zap.String("rid", rid))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	author, content, stars, ownerReview := review.Author, review.Content, review.Stars, review.OwnerReview

	if ownerReview {
		err := api.AuthzEntityPermissionCheck(
			d.Context,
			d.Auth,
			targetType,
			targetId,
			perms.EntityManageOwnerReviews,
		)

		if err != nil {
			return resp.Forbidden("Entity permission checks failed: " + err.Error())
		}
	} else {
		if d.Auth.TargetType != api.TargetTypeUser {
			return resp.Forbidden("Only users may delete non-owner reviews")
		} else if d.Auth.TargetType == api.TargetTypeUser {
			if author != d.Auth.ID {
				return resp.Forbidden("You are not the author of this review")
			}
		} else {
			return resp.ErrBody("Unreachable condition reached!", "Unreachable condition reached!", nil)
		}
	}

	err = q.DeleteReviewByID(d.Context, ridUUID)

	if err != nil {
		return resp.Err("Failed to delete review [db exec]", err, zap.String("rid", rid))
	}

	err = drivers.Send(drivers.With{
		Data: events.WebhookDeleteReviewData{
			ReviewID:    rid,
			Content:     content,
			Stars:       stars,
			OwnerReview: ownerReview,
		},
		UserID:     d.Auth.ID,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType), zap.String("user_id", d.Auth.ID), zap.String("review_id", rid))
	}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				state.Logger.Error("Panic while triggering review GC", zap.Any("panic", rec), zap.String("target_id", targetId), zap.String("target_type", targetType))
			}
		}()

		if err := assets.GCTrigger(targetId, targetType); err != nil {
			state.Logger.Error("Failed to trigger GC: ", zap.Error(err))
		}
	}()

	return uapi.DefaultResponse(http.StatusNoContent)
}
