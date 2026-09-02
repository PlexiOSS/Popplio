package add_review

import (
	"net/http"
	"time"

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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/ratelimit"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

var compiledMessages = uapi.CompileValidationErrors(types.CreateReview{})

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Review",
		Description: "Creates a new review for an entity. A user may have only one `root review` per entity. Triggers a garbage collection step to remove any orphaned reviews afterwards. Note that non-users can only create an 'owner review'. Returns 204 on success",
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
		},
		Req:  types.CreateReview{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Minute,
		MaxRequests: 2,
		Bucket:      "review",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "review"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	var payload types.CreateReview

	hresp, ok := uapi.MarshalReqWithHeaders(r, &payload, limit.Headers())

	if !ok {
		return hresp
	}

	err = state.Validator.Struct(payload)

	if err != nil {
		errors := err.(validator.ValidationErrors)
		return uapi.ValidatorErrorResponse(compiledMessages, errors)
	}

	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	q := db.New(state.Pool)

	switch targetType {
	case "bot":
		count, err := q.CountBotByID(d.Context, targetId)

		if err != nil {
			return resp.Err("Failed to query bot count [db count]", err, zap.String("bot_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Bot not found")
		}
	case "server":
		count, err := q.CountServerByID(d.Context, targetId)

		if err != nil {
			return resp.Err("Failed to query server count [db count]", err, zap.String("server_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Server not found")
		}
	case "team":
		count, err := q.CountTeamByID(d.Context, targetId)

		if err != nil {
			return resp.Err("Failed to query team count [db count]", err, zap.String("team_id", targetId))
		}

		if count == 0 {
			return resp.BadRequest("Team not found")
		}
	case "server_template":
		exists, err := q.CountServerTemplateByID(d.Context, targetId)

		if err != nil {
			return resp.Err("Failed to query server template count [db count]", err, zap.String("template_id", targetId))
		}

		if !exists {
			return resp.BadRequest("Server template not found")
		}
	default:
		return resp.Status(http.StatusNotImplemented, "Support for this target type has not been implemented yet")
	}

	if d.Auth.TargetType != api.TargetTypeUser && !payload.OwnerReview {
		return resp.Forbidden("Only users may create non-owner reviews")
	}

	if payload.OwnerReview {
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
	}

	if payload.ParentID == "" {
		count, err := q.CountRootReview(d.Context, db.CountRootReviewParams{
			Author:     d.Auth.ID,
			TargetID:   targetId,
			TargetType: targetType,
		})

		if err != nil {
			return resp.Err("Failed to query root review count [db count]", err, zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
		}

		if count > 0 {
			return resp.Conflict("You have already made a root review for this " + targetType)
		}
	}

	var parentID pgtype.UUID
	if payload.ParentID != "" {
		if err := parentID.Scan(payload.ParentID); err != nil {
			return resp.BadRequest("Parent review not found")
		}

		count, err := q.CountReviewByID(d.Context, parentID)

		if err != nil {
			return resp.Err("Failed to query parent review count [db count]", err, zap.String("parent_id", payload.ParentID))
		}

		if count == 0 {
			return resp.BadRequest("Parent review not found")
		}

		nest, err := assets.Nest(d.Context, payload.ParentID)

		if err != nil {
			return resp.Err("Nesting engine failed unexpectedly", err, zap.String("parent_id", payload.ParentID))
		}

		if nest > 2 {
			return resp.BadRequest("Maximum nesting for reviews reached")
		}
	}

	reviewIdUUID, err := q.InsertReview(d.Context, db.InsertReviewParams{
		Author:      d.Auth.ID,
		TargetID:    targetId,
		TargetType:  targetType,
		Content:     payload.Content,
		Stars:       payload.Stars,
		ParentID:    parentID,
		OwnerReview: payload.OwnerReview,
	})

	if err != nil {
		return resp.Err("Failed to insert review", err, zap.String("author", d.Auth.ID), zap.String("target_id", targetId), zap.String("target_type", targetType))
	}

	reviewId := uuid.UUID(reviewIdUUID.Bytes).String()

	err = drivers.Send(drivers.With{
		Data: events.WebhookNewReviewData{
			ReviewID:    reviewId,
			Content:     payload.Content,
			Stars:       payload.Stars,
			OwnerReview: payload.OwnerReview,
		},
		UserID:     d.Auth.ID,
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to send webhook", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType), zap.String("user_id", d.Auth.ID), zap.String("review_id", reviewId))
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
