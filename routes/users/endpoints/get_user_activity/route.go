// Package get_user_activity implements GET /users/{id}/activity — "Get User
// Activity".
//
// Returns a user's voter leaderboard positions, most-voted bots and servers,
// and recent reviews
package get_user_activity

import (
	"net/http"
	"sync"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"
	"popplio/votes"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/dovewing"
	"github.com/PlexiOSS/Keel/uapi"
)

const (
	topVotedLimit = 5
	reviewsLimit  = 5
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get User Activity",
		Description: "Returns a user's voter leaderboard positions (this month and all time), the listed bots and servers they've upvoted most, and their most recent reviews. Only listed, non-NSFW bots and servers are included.",
		Params: []docs.Parameter{
			{
				Name:        "id",
				Description: "The user's ID",
				Required:    true,
				In:          "path",
				Schema:      docs.IdSchema,
			},
		},
		Resp: types.UserActivity{},
	}
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	userID := chi.URLParam(r, "id")

	if userID == "" {
		return resp.BadRequest("id must be specified")
	}

	res := types.UserActivity{
		TopVoted: []types.VoterLeaderboardFavorite{},
		Reviews:  []types.UserActivityReview{},
	}

	q := db.New(state.Pool)
	g, ctx := errgroup.WithContext(d.Context)

	g.Go(func() error {
		rank, err := votes.LeaderboardRank(ctx, votes.LeaderboardPeriodMonth, userID)
		res.Month = rank
		return err
	})

	g.Go(func() error {
		rank, err := votes.LeaderboardRank(ctx, votes.LeaderboardPeriodAll, userID)
		res.AllTime = rank
		return err
	})

	g.Go(func() error {
		rows, err := q.GetUserTopVoted(ctx, db.GetUserTopVotedParams{Author: userID, Lim: topVotedLimit})

		if err != nil {
			return err
		}

		for _, row := range rows {
			res.TopVoted = append(res.TopVoted, types.VoterLeaderboardFavorite{
				TargetType: row.TargetType,
				TargetID:   row.TargetID,
				Name:       row.ServerName,
				Avatar:     row.ServerAvatar,
				Vanity:     row.Vanity,
				Votes:      int(row.VoteCount),
			})
		}

		return nil
	})

	g.Go(func() error {
		rows, err := q.GetUserRecentReviews(ctx, db.GetUserRecentReviewsParams{Author: userID, Lim: reviewsLimit})

		if err != nil {
			return err
		}

		for _, row := range rows {
			res.Reviews = append(res.Reviews, types.UserActivityReview{
				ID:        row.ID.String(),
				Stars:     int(row.Stars),
				Content:   row.Content,
				CreatedAt: row.CreatedAt.Time,
				Target: &types.VoterLeaderboardFavorite{
					TargetType: row.TargetType,
					TargetID:   row.TargetID,
					Name:       row.ServerName,
					Avatar:     row.ServerAvatar,
					Vanity:     row.Vanity,
				},
			})
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return resp.Err("Failed to get user activity", err)
	}

	targets := make([]*types.VoterLeaderboardFavorite, 0, len(res.TopVoted)+len(res.Reviews))

	for i := range res.TopVoted {
		targets = append(targets, &res.TopVoted[i])
	}

	for i := range res.Reviews {
		targets = append(targets, res.Reviews[i].Target)
	}

	var wg sync.WaitGroup

	for _, target := range targets {
		if target.TargetType != "bot" {
			continue
		}

		wg.Go(func() {
			bot, err := dovewing.GetUser(d.Context, target.TargetID, state.DovewingPlatformDiscord)

			if err != nil {
				target.Name = target.TargetID
				return
			}

			target.Name = bot.DisplayName

			if target.Name == "" {
				target.Name = bot.Username
			}

			target.Avatar = bot.Avatar
		})
	}

	wg.Wait()

	return uapi.HttpResponse{
		Json: res,
	}
}
