package get_app_list

import (
	"encoding/json"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/routes/staff/assets"
	"popplio/state"
	"popplio/types"

	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/jackc/pgx/v5/pgtype"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Staff: Get Application List",
		Description: "Gets all applications returning a list of apps.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The ID of the user to get the applications for. If not specified, all applications will be returned.",
				In:          "query",
				Required:    false,
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.AppListResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var err error
	d.Auth.ID, err = assets.EnsurePanelAuth(d.Context, r)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	staffPerms, err := perms.StaffPerms(d.Context, d.Auth.ID)

	if err != nil {
		return resp.Status(http.StatusFailedDependency, err.Error())
	}

	if !staffPerms.Has(perms.StaffViewApps) {
		return resp.Forbidden("You do not have permission to view apps.")
	}

	userId := r.URL.Query().Get("user_id")

	userIDFilter := pgtype.Text{}
	if userId != "" {
		userIDFilter = pgtype.Text{String: userId, Valid: true}
	}

	rows, err := db.New(state.Pool).GetAppsList(d.Context, userIDFilter)

	if err != nil {
		return resp.Err("Failed to fetch application list [db fetch]", err)
	}

	app := make([]types.AppResponse, len(rows))
	for i, row := range rows {
		var questions []types.Question
		if err := json.Unmarshal(row.Questions, &questions); err != nil {
			return resp.Err("Failed to parse application questions [json]", err)
		}

		var reviewFeedback *string
		if row.ReviewFeedback.Valid {
			reviewFeedback = &row.ReviewFeedback.String
		}

		app[i] = types.AppResponse{
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

	for i := range app {
		user, err := dovewing.GetUser(d.Context, app[i].UserID, state.DovewingPlatformDiscord)

		if err != nil {
			return resp.Err("Failed to fetch application list [user fetch]", err, zap.String("userId", app[i].UserID))
		}

		app[i].User = user
	}

	return uapi.HttpResponse{
		Json: types.AppListResponse{
			Apps: app,
		},
	}
}
