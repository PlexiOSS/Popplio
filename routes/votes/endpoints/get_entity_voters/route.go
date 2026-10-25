// Package get_entity_voters implements GET
// /{target_type}/{target_id}/votes/voters — "Get Entity Voters".
//
// Returns the users who have upvoted an entity with their vote counts, most
// active first
package get_entity_voters

import (
	"net/http"
	"strconv"
	"sync"

	"popplio/api/resp"
	"popplio/db"
	"popplio/listing"
	"popplio/state"
	"popplio/types"
	"popplio/validators"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const (
	defaultLimit = 24
	maxLimit     = 100
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Entity Voters",
		Description: "Returns the users who have upvoted an entity, with their upvotes since the last vote reset and of all time, ordered by current votes, then all-time votes, then most recent vote. Voters whose profile can't be resolved are left out.",
		Params: []docs.Parameter{
			{
				Name:        "target_type",
				Description: "The target type of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "target_id",
				Description: "The target ID of the entity",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "limit",
				Description: "How many voters to return. Defaults to 24, capped at 100.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.EntityVoterList{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	targetId := chi.URLParam(r, "target_id")
	targetType := validators.NormalizeTargetType(chi.URLParam(r, "target_type"))

	if targetId == "" || targetType == "" {
		return resp.BadRequest("Both target_id and target_type must be specified")
	}

	limit := defaultLimit

	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)

		if err != nil || parsed <= 0 {
			return resp.BadRequest("limit must be a positive integer")
		}

		limit = int(min(parsed, maxLimit))
	}

	visible, err := listing.Visible(d.Context, d.Auth, targetType, targetId)

	if err != nil {
		return resp.Err("Failed to check listing status", err)
	}

	if !visible {
		return uapi.DefaultResponse(http.StatusNotFound)
	}

	rows, err := db.New(state.Pool).GetEntityVoters(d.Context, db.GetEntityVotersParams{
		TargetID:   targetId,
		TargetType: targetType,
		Lim:        int32(limit),
	})

	if err != nil {
		return resp.Err("Failed to get entity voters", err, zap.String("targetId", targetId), zap.String("targetType", targetType))
	}

	res := types.EntityVoterList{Voters: []types.EntityVoter{}}

	if len(rows) > 0 {
		res.Total = int(rows[0].TotalVoters)
	}

	voters := make([]types.EntityVoter, len(rows))

	var wg sync.WaitGroup

	for i, row := range rows {
		voters[i] = types.EntityVoter{
			Votes:       int(row.CurrentVotes),
			TotalVotes:  int(row.TotalVotes),
			LastVotedAt: row.LastVotedAt.Time,
		}

		wg.Go(func() {
			user, err := dovewing.GetUser(d.Context, row.Author, state.DovewingPlatformDiscord)

			if err != nil {
				state.Logger.Warn("Leaving voter out of entity voters: user lookup failed", zap.String("user_id", row.Author), zap.Error(err))
				return
			}

			voters[i].User = user
		})
	}

	wg.Wait()

	for _, v := range voters {
		if v.User != nil {
			res.Voters = append(res.Voters, v)
		}
	}

	return uapi.HttpResponse{
		Json: res,
	}
}
