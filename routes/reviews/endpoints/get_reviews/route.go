// Package get_reviews implements GET /{target_type}/{target_id}/reviews —
// "Get Reviews".
//
// Gets the reviews of a bot by its ID or vanity.
package get_reviews

import (
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

var (
	reviewColsArr = dbutil.GetCols(types.Review{})
	reviewCols    = strings.Join(reviewColsArr, ",")
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

	rows, err := state.Pool.Query(d.Context, "SELECT "+reviewCols+" FROM reviews WHERE target_id = $1 AND target_type = $2 ORDER BY created_at ASC", targetId, targetType)

	if err != nil {
		state.Logger.Error("Failed to query reviews [db query]", zap.Error(err), zap.String("target_id", targetId), zap.String("target_type", targetType))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	reviews, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Review])

	if err != nil {
		return resp.Err("Failed to query reviews [collect]", err, zap.String("target_id", targetId), zap.String("target_type", targetType))
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
