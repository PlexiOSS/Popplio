// Package get_voter_leaderboard implements GET /votes/leaderboard — "Get
// Voter Leaderboard".
//
// Returns the most active voters, all-time, by total upvotes cast
package get_voter_leaderboard

import (
	"net/http"
	"strconv"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	"golang.org/x/sync/errgroup"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Voter Leaderboard",
		Description: "Returns the most active voters, all-time, by total upvotes cast",
		Params: []docs.Parameter{
			{
				Name:        "limit",
				Description: "How many voters to return. Defaults to 10, capped at 50.",
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
		parsed, err := strconv.Atoi(raw)

		if err != nil || parsed <= 0 {
			return resp.BadRequest("limit must be a positive integer")
		}

		limit = min(parsed, maxLimit)
	}

	rows, err := db.New(state.Pool).GetTopVoters(d.Context, int32(limit))

	if err != nil {
		return resp.Err("Failed to fetch voter leaderboard", err)
	}

	entries := make([]types.VoterLeaderboardEntry, len(rows))
	g, ctx := errgroup.WithContext(d.Context)

	for i, row := range rows {
		entries[i] = types.VoterLeaderboardEntry{Votes: int(row.VoteCount)}

		g.Go(func() error {
			user, err := dovewing.GetUser(ctx, row.Author, state.DovewingPlatformDiscord)

			if err != nil {
				return err
			}

			entries[i].User = user

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return resp.Err("Failed to resolve voter leaderboard users", err)
	}

	return uapi.HttpResponse{
		Json: entries,
	}
}
