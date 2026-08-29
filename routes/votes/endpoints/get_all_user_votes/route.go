// Package get_all_user_votes implements GET
// /users/{uid}/{target_type}/{target_id}/votes/@all — "Get All User Votes".
//
// Gets all votes (paginated by 10) of a user on an entity. This endpoint is
// currently public as the same data can be found through #vote-logs in
// discord. Note that for compatibility, a trailing 's' is removed
package get_all_user_votes

import (
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/pagination"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 5

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get All User Votes",
		Description: "Gets all votes (paginated by 10) of a user on an entity. This endpoint is currently public as the same data can be found through #vote-logs in discord. Note that for compatibility, a trailing 's' is removed",
		Resp:        types.PagedResult[[]types.EntityVote]{},
		RespName:    "PagedResultUserVote",
		Params: []docs.Parameter{
			{
				Name:        "uid",
				Description: "The users ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
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
	uid := chi.URLParam(r, "uid")
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if uid == "" || targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	pageNum, err := pagination.Parse(r)

	if err != nil {
		return resp.BadRequest("Invalid page number")
	}

	limit := perPage
	offset := (pageNum - 1) * perPage

	q := db.New(state.Pool)

	rows, err := q.GetUserEntityVotesPage(d.Context, db.GetUserEntityVotesPageParams{
		TargetID:   targetId,
		TargetType: targetType,
		Author:     uid,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	ev := make([]types.EntityVote, len(rows))
	for i, row := range rows {
		ev[i] = types.EntityVote{
			ITag:       row.Itag,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			AuthorID:   row.Author,
			Upvote:     row.Upvote,
			Void:       row.Void,
			VoidReason: row.VoidReason,
			VoidedAt:   pgtype.Timestamp{Time: row.VoidedAt.Time, Valid: row.VoidedAt.Valid},
			CreatedAt:  row.CreatedAt.Time,
			VoteNum:    int(row.VoteNum),
			Credit:     row.CreditRedeem,
			Immutable:  row.Immutable,
		}
	}

	count, err := q.CountUserEntityVotes(d.Context, db.CountUserEntityVotesParams{
		TargetID:   targetId,
		TargetType: targetType,
		Author:     uid,
	})

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("userId", uid), zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	data := types.PagedResult[[]types.EntityVote]{
		Count:   uint64(count),
		PerPage: perPage,
		Results: ev,
	}

	return uapi.HttpResponse{
		Json: data,
	}
}
