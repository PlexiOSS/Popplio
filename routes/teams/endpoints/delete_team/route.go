package delete_team

import (
	"net/http"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Delete Team",
		Description: "Deletes the team. Requires the 'Owner' permission. Returns a 204 on success",
		Params: []docs.Parameter{
			{
				Name:        "tid",
				Description: "Team ID",
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

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamId); err != nil {
		return resp.BadRequest("Invalid team ID")
	}

	tx, err := state.Pool.Begin(d.Context)

	if err != nil {
		return resp.Err("Error beginning transaction", err, zap.String("tid", teamId))
	}

	q := db.New(tx)

	// Without this, a bot/server can be added to this team (add_bot/
	// add_server, which take the same lock before inserting) in the window
	// between the count checks below passing at 0 and the DELETE FROM teams
	// below -- both bots.team_owner and servers.team_owner are ON DELETE
	// CASCADE, so that newly-added bot/server would be silently
	// cascade-deleted along with the team instead of the add either
	// blocking until this delete finishes or the delete correctly seeing a
	// nonzero count.
	if err := q.LockTeamOwnership(d.Context, teamId); err != nil {
		return resp.Err("Error acquiring team ownership lock", err, zap.String("tid", teamId))
	}

	botCount, err := q.CountBotsByTeamOwner(d.Context, teamUUID)

	if err != nil {
		return resp.Err("Error getting bot count [db count]", err, zap.String("tid", teamId))
	}

	if botCount > 0 {
		return resp.BadRequest("You cannot delete a team with bots in it")
	}

	serverCount, err := q.CountServersByTeamOwner(d.Context, teamUUID)

	if err != nil {
		return resp.Err("Error getting server count [db count]", err, zap.String("tid", teamId))
	}

	if serverCount > 0 {
		return resp.BadRequest("You cannot delete a team with servers in it")
	}

	err = q.DeleteTeamMembers(d.Context, teamUUID)

	if err != nil {
		return resp.Err("Error deleting team members", err, zap.String("tid", teamId))
	}

	err = q.DeleteTeam(d.Context, teamId)

	if err != nil {
		return resp.Err("Error deleting team", err, zap.String("tid", teamId))
	}

	err = tx.Commit(d.Context)

	if err != nil {
		return resp.Err("Error committing transaction", err, zap.String("tid", teamId))
	}

	return uapi.DefaultResponse(http.StatusNoContent)
}
