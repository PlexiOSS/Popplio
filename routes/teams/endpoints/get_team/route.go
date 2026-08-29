package get_team

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/teams/resolvers"
	"popplio/types"
	"popplio/votes"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"

	"github.com/go-chi/chi/v5"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Team",
		Description: "Gets a team by ID",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "Team ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "targets",
				Description: "Entities to get. Can be one of the following: `team_member`, `bot`, `server`. Comma-seperated",
				Required:    true,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.Team{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	id := chi.URLParam(r, "id")
	targetStr := r.URL.Query().Get("targets")
	targets := strings.Split(targetStr, ",")

	if _, err := uuid.Parse(id); err != nil {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	q := db.New(state.Pool)

	row, err := q.GetTeamByID(d.Context, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	if err != nil {
		return resp.Err("Error querying team [db query]", err, zap.String("id", id))
	}

	var extraLinks []types.Link
	if err := json.Unmarshal(row.ExtraLinks, &extraLinks); err != nil {
		return resp.Err("Error parsing team extra_links [json]", err, zap.String("id", id))
	}

	team := types.Team{
		ID:               row.ID,
		Name:             row.Name,
		Short:            row.Short,
		Tags:             row.Tags,
		VoteBanned:       row.VoteBanned,
		ApproximateVotes: int(row.ApproximateVotes),
		ExtraLinks:       extraLinks,
		NSFW:             row.Nsfw,
		VanityRef:        row.VanityRef,
		Service:          row.Service,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}

	if team.Tags == nil {
		team.Tags = []string{}
	}
	if team.ExtraLinks == nil {
		team.ExtraLinks = []types.Link{}
	}

	team.Entities, err = resolvers.GetTeamEntities(d.Context, id, targets)

	if err != nil {
		return resp.Err("Error resolving team entities", err, zap.String("id", id))
	}

	code, err := q.GetVanityCodeByItag(d.Context, team.VanityRef)

	if err != nil {
		return resp.Err("Error while getting bot vanity code [db fetch]", err, zap.String("id", id), zap.String("teamId", team.ID))
	}

	team.Vanity = code

	team.Votes, err = votes.EntityGetVoteCount(d.Context, state.Pool, id, "team")

	if err != nil {
		return resp.Err("Error while getting team vote count", err, zap.String("id", id))
	}

	return uapi.HttpResponse{
		Json: team,
	}
}
