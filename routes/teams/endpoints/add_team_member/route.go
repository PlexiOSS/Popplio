package add_team_member

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
		Summary:     "Add Team Member",
		Description: "Adds a member to a team. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Req:  types.AddTeamMember{},
		Resp: types.ApiError{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	var teamId = chi.URLParam(r, "tid")

	var payload types.AddTeamMember

	hresp, ok := uapi.MarshalReq(r, &payload)

	if !ok {
		return hresp
	}

	managerPerms, err := teams.GetEntityPerms(d.Context, d.Auth.ID, "team", teamId)

	if err != nil {
		state.Logger.Error("Error getting user perms", zap.Error(err), zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
		return resp.BadRequest("Error getting user perms: " + err.Error())
	}

	for _, perm := range payload.Perms {
		if !teams.IsValidPerm(perm) {
			return resp.BadRequest("Invalid permission: " + perm)
		}
	}

	newPermsResolved := perms.Entity.ResolveStrings(payload.Perms)

	if err = perms.CheckPatch(managerPerms, perms.Entity.NewSet(), newPermsResolved); err != nil {
		return resp.Forbidden("You do not have permission to give out permissions: " + err.Error())
	}

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamId); err != nil {
		return resp.BadRequest("Invalid team ID")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error starting transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	defer tx.Rollback(d.Context)

	q := db.New(tx)

	userExists, err := q.UserExistsCheck(d.Context, payload.UserID)

	if err != nil {
		return resp.Err("Error checking if user exists", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	if !userExists {
		return resp.BadRequest("User must login here at least once before you can add them")
	}

	memberExists, err := q.TeamMemberExists(d.Context, db.TeamMemberExistsParams{
		TeamID: teamUUID,
		UserID: payload.UserID,
	})

	if err != nil {
		return resp.Err("Error checking if user is already a member", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	if memberExists {
		return resp.BadRequest("User is already a member of this team")
	}

	err = q.InsertTeamMember(d.Context, db.InsertTeamMemberParams{
		TeamID: teamUUID,
		UserID: payload.UserID,
		Flags:  payload.Perms,
	})

	if err != nil {
		return resp.Err("Error adding member", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("uid", d.Auth.ID), zap.String("tid", teamId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
