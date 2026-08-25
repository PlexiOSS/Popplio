// Package get_user_perms implements GET /users/{id}/perms — "Get User
// Perms".
//
// Gets a users permissions by ID
package get_user_perms

import (
	"errors"
	"net/http"
	"strings"

	"popplio/api/resp"

	"github.com/PlexiOSS/Keel/dbutil"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

var (
	userPermColsArr = dbutil.GetCols(types.UserPerm{})
	userPermCols    = strings.Join(userPermColsArr, ",")
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

	row, err := state.Pool.Query(d.Context, "SELECT "+userPermCols+" FROM users WHERE user_id = $1", id)

	if err != nil {
		state.Logger.Error("Failed to get user perms", zap.Error(err), zap.String("user_id", id))
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	up, err := pgx.CollectOneRow(row, pgx.RowToStructByName[types.UserPerm])

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Failed to get user perms", err, zap.String("user_id", id))
	}

	user, err := dovewing.GetUser(d.Context, id, state.DovewingPlatformDiscord)

	if err != nil {
		return resp.Err("Failed to get user perms", err, zap.String("user_id", id))
	}

	up.User = user

	// Fetch staff status
	var positions int

	err = state.Pool.QueryRow(d.Context, "SELECT cardinality(positions) FROM staff_members WHERE user_id = $1", user.ID).Scan(&positions)

	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return resp.Err("Error while getting staff status", err, zap.String("userID", user.ID))
	}

	up.Staff = positions > 0

	return uapi.HttpResponse{
		Json: up,
	}
}
