// Package get_user_perms implements GET /users/{id}/perms — "Get User
// Perms".
//
// Gets a users permissions by ID
package get_user_perms

import (
	"errors"
	"net/http"

	"popplio/api/resp"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Perms",
		Description: "Gets a users permissions by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "User ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserPerm{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")

	q := db.New(state.Pool)

	row, err := q.GetUserPerm(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		state.Logger.Error("Failed to get user perms", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	up := types.UserPerm{
		ID:                    id,
		Experiments:           row.Experiments,
		Banned:                row.Banned,
		CaptchaSponsorEnabled: row.CaptchaSponsorEnabled,
		VoteBanned:            row.VoteBanned,
	}

	user, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Failed to get user perms", err, zap.String("user_id", id))
	}

	up.User = user

	// Fetch staff status
	positions, err := q.GetStaffPositionCount(d.Context, user.ID)

	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return resp.Err("Error while getting staff status", err, zap.String("userID", user.ID))
	}

	up.Staff = positions > 0

	return uapi.HttpResponse{
		Json: up,
	}
}
