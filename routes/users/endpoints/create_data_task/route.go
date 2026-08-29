// Package create_data_task implements POST /users/{id}/data — "Create Data
// Task".
//
// Creates a data task for a user (delete or request). Returns the task id if
// this is successful.
package create_data_task

import (
	"net/http"
	"strings"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/routes/users/endpoints/create_data_task/assets"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/PlexiOSS/Keel/crypto"
	"github.com/PlexiOSS/Keel/ratelimit"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const dataTaskExpiryTime = time.Hour * 1

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Create Data Task",
		Description: "Creates a data task for a user (delete or request). Returns the task id if this is successful.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "delete",
				Description: "Whether we should do a Data Deletion or a Data Request",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.TaskCreateResponse{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	reqType := r.URL.Query().Get("delete")

	if reqType != "true" && reqType != "false" {
		return resp.BadRequest("delete must be ether 'true' or 'false'")
	}

	limit, err := ratelimit.Ratelimit{
		Expiry:      1 * time.Hour,
		MaxRequests: 50,
		Bucket:      "data_request",
	}.Limit(d.Context, r)

	if err != nil {
		return resp.Err("Error while ratelimiting", err, zap.String("bucket", "data_request"))
	}

	if limit.Exceeded {
		return resp.RateLimited(limit)
	}

	taskName := "data_request"

	if reqType == "true" {
		taskName = "data_delete"
	}

	remoteIp := strings.Split(strings.ReplaceAll(r.Header.Get("X-Forwarded-For"), " ", ""), ",")

	taskKey := crypto.RandString(128)

	allowUnauthenticated := (taskName == "data_delete") // Only data deletions need unauthenticated access to task data

	expiry := pgtype.Interval{
		Microseconds: int64(dataTaskExpiryTime / time.Microsecond),
		Valid:        true,
	}

	taskId, err := db.New(state.Pool).InsertTask(d.Context, db.InsertTaskParams{
		TaskName: taskName,
		TaskKey:  pgtype.Text{String: taskKey, Valid: true},
		ForUser:  pgtype.Text{String: d.Auth.ID, Valid: true},
		Expiry:   expiry,
		Output: map[string]any{
			"meta": map[string]any{
				"request_ip": remoteIp[0],
			},
		},
		AllowUnauthenticated: allowUnauthenticated,
	})

	if err != nil {
		return resp.ErrBody("Error creating task", "Error creating task.", err)
	}

	go assets.DataTask(taskId, taskName, d.Auth.ID, remoteIp[0])

	return uapi.HttpResponse{
		Json: types.TaskCreateResponse{
			TaskID: taskId,
			TaskKey: pgtype.Text{
				Valid:  true,
				String: taskKey,
			},
			TaskName:             taskName,
			Expiry:               expiry,
			AllowUnauthenticated: allowUnauthenticated,
		},
	}
}
