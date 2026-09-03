package edit_team_member

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

var globalOwner = perms.EntityOwner

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Edit Team Member Permissions",
		Description: "Edits a members permissions on a team. Returns a 204 on success",
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
				Description: "Team Member ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.EditTeamMember{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")
	var userId = chi.URLParam(r, "mid")

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamId); err != nil {
		return resp.BadRequest("Invalid team ID")
	}

	count, err := db.New(state.Pool).CountTeamMembership(d.Context, db.CountTeamMembershipParams{
		TeamID:   teamUUID,
		UserID:   d.Auth.ID,
		UserID_2: userId,
	})

	if err != nil {
		return resp.ErrDetail("Error checking if user is on team", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	if d.Auth.ID != userId {
		if count != 2 {
			return resp.BadRequest("Either the manager or the user is not on this team")
		}
	} else if count != 1 {
		return resp.BadRequest("User is not on this team")
	}

	var payload types.EditTeamMember

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		return resp.BadRequest("Error getting user perms.")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error starting transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	if payload.Perms != nil {
		currentUserPerms, err := teams.GetEntityPerms(d.Context, userId, "team", teamId)

		if err != nil {
			return resp.ErrDetail("Error getting old perms", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}

		for _, perm := range *payload.Perms {
			if !teams.IsValidPerm(perm) {
				return resp.BadRequest("Invalid permission: " + perm)
			}
		}

		newPermsResolved := perms.Entity.ResolveStrings(*payload.Perms)

		if err = perms.CheckPatch(managerPerms, currentUserPerms, newPermsResolved); err != nil {
			return resp.Forbidden("You do not have permission to set these permissions.")
		}

		if !newPermsResolved.Has(globalOwner) && currentUserPerms.Has(globalOwner) {
			if err := q.LockTeamOwnership(d.Context, teamId); err != nil {
				return resp.Err("Error acquiring team ownership lock", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			}

			ownerCount, err := q.CountTeamOwnersWithFlag(d.Context, db.CountTeamOwnersWithFlagParams{
				TeamID: teamUUID,
				Flags:  []string{globalOwner.String()},
			})

			if err != nil {
				return resp.Err("Error getting owner count", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			}

			if ownerCount < 2 {
				return resp.BadRequest("There needs to be one other global owner before you can remove yourself from owner")
			}
		}

		err = q.UpdateTeamMemberFlags(d.Context, db.UpdateTeamMemberFlagsParams{
			Flags:  *payload.Perms,
			TeamID: teamUUID,
			UserID: userId,
		})

		if err != nil {
			return resp.Err("Error updating perms", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}
	}

	if payload.Mentionable != nil {
		if d.Auth.ID != userId {
			if !managerPerms.Has(perms.EntityEditMembers) {
				return resp.Forbidden("You do not have permission to edit this member")
			}
		}

		err = q.UpdateTeamMemberMentionable(d.Context, db.UpdateTeamMemberMentionableParams{
			Mentionable: *payload.Mentionable,
			TeamID:      teamUUID,
			UserID:      userId,
		})

		if err != nil {
			return resp.Err("Error updating mentionable", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}
	}

	if payload.DataHolder != nil {
		if !managerPerms.Has(globalOwner) {
			return resp.Forbidden("Only global owners can set a data holder")
		}

		if !*payload.DataHolder {
			dataHolderCount, err := q.CountTeamDataHolders(d.Context, db.CountTeamDataHoldersParams{
				TeamID:     teamUUID,
				DataHolder: true,
				UserID:     userId,
			})

			if err != nil {
				return resp.Err("Error getting data holder count", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
			}

			if dataHolderCount == 0 {
				return resp.BadRequest("There needs to be one other data holder before you can remove someone from data holder")
			}
		}

		err = q.UpdateTeamMemberDataHolder(d.Context, db.UpdateTeamMemberDataHolderParams{
			DataHolder: *payload.DataHolder,
			TeamID:     teamUUID,
			UserID:     userId,
		})

		if err != nil {
			return resp.Err("Error updating data holder", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
		}
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId), zap.String("mid", userId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
