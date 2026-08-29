// Package get_task implements GET /users/{id}/tasks/{tid} — "Get Task".
//
// Gets a task. Returns the task data if this is successful
package get_task

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Task",
		Description: "Gets a task. Returns the task data if this is successful",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "tid",
				Description: "The task ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "task_key",
				Description: "The task key if required. This is used to authenticate the request.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Task{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	// Check that the user owns the task
	taskId := chi.URLParam(r, "tid")
	userId := chi.URLParam(r, "id")

	if taskId == "" {
		return resp.BadRequest("task id is required")
	}

	q := db.New(state.Pool)

	// Delete expired tasks first
	err := q.DeleteExpiredTasks(d.Context)

	if err != nil {
		return resp.Err("Failed to delete expired tasks [db delete]", err)
	}

	row, err := q.GetTask(d.Context, taskId)

	if errors.Is(err, pgx.ErrNoRows) {
		return resp.NotFound("Task not found")
	}

	if err != nil {
		return resp.Err("Failed to fetch task [db fetch]", err)
	}

	task := types.Task{
		TaskId:               row.TaskID,
		TaskKey:              row.TaskKey,
		AllowUnauthenticated: row.AllowUnauthenticated,
		TaskName:             row.TaskName,
		Output:               row.Output,
		Statuses:             row.Statuses,
		ForUser:              row.ForUser,
		Expiry:               row.Expiry,
		State:                row.State,
		CreatedAt:            pgtype.Timestamptz{Time: row.CreatedAt.Time, Valid: row.CreatedAt.Valid},
	}

	if task.TaskKey.Valid {
		if task.TaskKey.String != r.URL.Query().Get("task_key") {
			return resp.Unauthorized("Invalid task key")
		}
	}

	if task.AllowUnauthenticated {
		d.Auth.ID = userId
	} else if d.Auth.ID == "" {
		return resp.Unauthorized("You must be authenticated to access this task")
	}

	if task.ForUser.Valid {
		if task.ForUser.String != d.Auth.ID {
			return resp.Forbidden("This task is not owned by your user account!")
		}
	}

	return uapi.HttpResponse{
		Status: http.StatusOK,
		Json:   task,
	}
}
