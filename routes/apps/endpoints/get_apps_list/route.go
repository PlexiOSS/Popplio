// Package get_apps_list implements GET /users/{user_id}/apps — "Get
// Application List".
//
// Gets all applications of the user returning a list of apps.
package get_apps_list

import (
	"encoding/json"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Application List",
		Description: "Gets all applications of the user returning a list of apps.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to use.",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AppListResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	rows, err := db.New(state.Pool).GetUserApps(d.Context, d.Auth.ID)

	if err != nil {
		state.Logger.Error("Failed to fetch application list [collection]", zap.String("userId", d.Auth.ID), zap.Error(err))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	apps := make([]types.AppResponse, len(rows))
	for i, row := range rows {
		var questions []types.Question
		if err := json.Unmarshal(row.Questions, &questions); err != nil {
			return resp.Err("Failed to parse application questions [json]", err, zap.String("userId", d.Auth.ID))
		}

		var reviewFeedback *string
		if row.ReviewFeedback.Valid {
			reviewFeedback = &row.ReviewFeedback.String
		}

		apps[i] = types.AppResponse{
			AppID:          row.AppID,
			UserID:         row.UserID,
			Questions:      questions,
			Answers:        row.Answers,
			State:          row.State,
			CreatedAt:      row.CreatedAt.Time,
			Position:       row.Position,
			ReviewFeedback: reviewFeedback,
		}
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: apps,
		},
	}
}
