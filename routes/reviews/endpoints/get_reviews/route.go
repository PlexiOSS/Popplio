package get_reviews

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Reviews",
		Description: "Gets the reviews of a bot by its ID or vanity.",
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
		Resp: types.ReviewList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	rows, err := db.New(state.Pool).GetReviews(d.Context, db.GetReviewsParams{
		TargetID:   targetId,
		TargetType: targetType,
	})

	if err != nil {
		state.Logger.Error("Failed to query reviews [db query]", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	reviews := make([]types.Review, len(rows))
	for i, row := range rows {
		reviews[i] = types.Review{
			ID:          row.ID,
			TargetType:  row.TargetType,
			TargetID:    row.TargetID,
			AuthorID:    row.Author,
			OwnerReview: row.OwnerReview,
			Content:     row.Content,
			Stars:       row.Stars,
			CreatedAt:   row.CreatedAt.Time,
			ParentID:    row.ParentID,
		}
	}

	for i := range reviews {
		user, err := dovewing.GetUser(d.Context, reviews[i].AuthorID, state.DovewingPlatformDiscord)

		if err != nil {
			state.Logger.Error("Failed to get user [dovewing]", zap.Error(err), zap.String("author_id", reviews[i].AuthorID))
			continue
		}

		reviews[i].Author = user
	}

	var allReviews = types.ReviewList{
		Reviews: reviews,
	}

	return uapi.HttpResponse{
		Json: allReviews,
	}
}
