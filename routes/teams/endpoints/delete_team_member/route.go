package delete_team_member

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/perms"
	"popplio/state"
	"popplio/teams"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team Member",
		Description: "Deletes a member from the team. Users can always delete themselves. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "mid",
				Description: "Member ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")
	var userId = chi.URLParam(r, "mid")

	userPerms, err := teams.GetEntityPerms(d.Context, userId, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return resp.BadRequest("Error getting user perms: " + err.Error())
	}

	if d.Auth.ID != userId {
		managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

		if err != nil {
			state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			return resp.BadRequest("Error getting user perms: " + err.Error())
		}

		if err := perms.CheckPatch(managerPerms, userPerms, perms.Entity.NewSet()); err != nil {
			return resp.Forbidden("You do not have permission to delete this member:" + err.Error())
		}
	}

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamId); err != nil {
		return resp.BadRequest("Invalid team ID")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error starting transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	if userPerms.Has(perms.EntityOwner) {
		if err := q.LockTeamOwnership(d.Context, teamId); err != nil {
			return resp.Err("Error acquiring team ownership lock", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}

		ownerCount, err := q.CountTeamOwnersWithFlag(d.Context, db.CountTeamOwnersWithFlagParams{
			TeamID: teamUUID,
			Flags:  []string{perms.EntityOwner.String()},
		})

		if err != nil {
			return resp.Err("Error getting owner count", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}

		if ownerCount < 2 {
			return resp.BadRequest("There needs to be one other global owner before you can remove yourself from owner")
		}
	}

	err = q.DeleteTeamMember(d.Context, db.DeleteTeamMemberParams{
		TeamID: teamUUID,
		UserID: userId,
	})

	if err != nil {
		return resp.Err("Error deleting member", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
