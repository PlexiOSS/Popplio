// Package get_voter_leaderboard_rank implements GET
// /votes/leaderboard/users/{user_id} — "Get Voter Leaderboard Rank".
//
// Returns one user's position on the voter leaderboard
package get_voter_leaderboard_rank

import (
	"errors"
	"net/http"

	"popplio/api/resp"
	"popplio/types"
	"popplio/votes"

	"github.com/go-chi/chi/v5"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Voter Leaderboard Rank",
		Description: "Returns one user's position on the voter leaderboard, counted the same way as Get Voter Leaderboard. A rank of 0 means the user has no counted upvotes in the period. Cached for 5 minutes.",
		Params: []docs.Parameter{
			{
				Name:        "user_id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
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
		Resp: types.VoterLeaderboardRank{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userID := chi.URLParam(r, "user_id")

	if userID == "" {
		return resp.BadRequest("user_id must be specified")
	}

	rank, err := votes.LeaderboardRank(d.Context, r.URL.Query().Get("period"), userID)

	if errors.Is(err, votes.ErrInvalidLeaderboardPeriod) {
		return resp.BadRequest(err.Error())
	}

	if err != nil {
		return resp.Err("Failed to fetch voter leaderboard rank", err)
	}

	return uapi.HttpResponse{
		Json: rank,
	}
}
