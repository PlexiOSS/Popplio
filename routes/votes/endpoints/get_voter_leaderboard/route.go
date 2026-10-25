// Package get_voter_leaderboard implements GET /votes/leaderboard — "Get
// Voter Leaderboard".
//
// Returns the most active voters by upvotes cast, all-time or since the
// last monthly vote reset
package get_voter_leaderboard

import (
	"errors"
	"net/http"
	"strconv"

	"popplio/api/resp"
	"popplio/types"
	"popplio/votes"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Voter Leaderboard",
		Description: "Returns the most active voters by upvotes cast, all-time or since the last monthly vote reset. Votes staff voided with a per-entity reset and votes from vote-banned or banned users are not counted. Cached for 5 minutes.",
		Params: []docs.Parameter{
			{
				Name:        "limit",
				Description: "How many voters to return. Defaults to 10, capped at 50.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
			{
				Name:        "period",
				Description: "all (default) for all-time, or month for votes since the last monthly vote reset.",
				Required:    false,
				In:          "query",
				Schema:      docs.IdSchema,
			},
		},
		Resp: []types.VoterLeaderboardEntry{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	limit := defaultLimit

	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)

		if err != nil || parsed <= 0 {
			return resp.BadRequest("limit must be a positive integer")
		}

		limit = int(min(parsed, maxLimit))
	}

	entries, err := votes.Leaderboard(d.Context, r.URL.Query().Get("period"), limit)

	if errors.Is(err, votes.ErrInvalidLeaderboardPeriod) {
		return resp.BadRequest(err.Error())
	}

	if err != nil {
		return resp.Err("Failed to fetch voter leaderboard", err)
	}

	return uapi.HttpResponse{
		Json: entries,
	}
}
