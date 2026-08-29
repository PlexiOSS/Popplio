// Package get_votes_user_list implements GET
// /{target_type}/{target_id}/votes/user-list — "Get Votes User List".
//
// Gets the full list of all users who have voted for the entity on Discord
// as a list of snowflakes. Note that for compatibility, a trailing 's' is
// removed. This method does not require authentication as it is easily
// publicly available through other means
package get_votes_user_list

import (
	"net/http"
	"strconv"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const perPage = 100

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Votes User List",
		Description: "Gets the full list of all users who have voted for the entity on Discord as a list of snowflakes. Note that for compatibility, a trailing 's' is removed. This method does not require authentication as it is easily publicly available through other means",
		Resp:        []string{},
		RespName:    "[]string",
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
				Description: "The page number (if pagination is desired, otherwise sends all results)",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	q := db.New(state.Pool)

	var ev []string
	var err error

	var page = r.URL.Query().Get("page")
	if page == "" {
		ev, err = q.GetVoteAuthors(d.Context, db.GetVoteAuthorsParams{TargetID: targetId, TargetType: targetType})
	} else {
		var pageNum uint64
		pageNum, err = strconv.ParseUint(page, 10, 32)

		if err != nil {
			return uapi.DefaultResponse(http.StatusBadRequest)
		}

		limit := perPage
		offset := (pageNum - 1) * perPage

		ev, err = q.GetVoteAuthorsPage(d.Context, db.GetVoteAuthorsPageParams{
			TargetID:   targetId,
			TargetType: targetType,
			Limit:      int32(limit),
			Offset:     int32(offset),
		})
	}

	if err != nil {
		return resp.Err("Failed to get user entity votes", err, zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	if ev == nil {
		ev = []string{}
	}

	return uapi.HttpResponse{
		Json: ev,
	}
}
