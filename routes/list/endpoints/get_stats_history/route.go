// Copyright (C) 2026 NodeByte LTD

package get_stats_history

import (
	"context"
	"net/http"
	"sync"
	"time"

	"popplio/api/resp"
	"popplio/db"
	"popplio/state"
	"popplio/types"

	docs "github.com/PlexiOSS/Keel/doclib"
	"github.com/PlexiOSS/Keel/uapi"
)

const cacheFor = 10 * time.Minute

var (
	cacheMu sync.Mutex
	cached  *types.StatsHistory
)

func Docs() *docs.Doc {
	return &docs.Doc{
		Summary:     "Get Stats History",
		Description: "All-time platform numbers and a month-by-month history of votes, reviews, new listings and sign-ups. Cached for 10 minutes.",
		Resp:        types.StatsHistory{},
	}
}

func Build(ctx context.Context, now time.Time) (*types.StatsHistory, error) {
	q := db.New(state.Pool)

	totals, err := q.GetStatsHistoryTotals(ctx)

	if err != nil {
		return nil, err
	}

	rows, err := q.GetMonthlyStatsHistory(ctx)

	if err != nil {
		return nil, err
	}

	out := &types.StatsHistory{
		VotesAllTime:   totals.VotesAllTime,
		VotesThisMonth: totals.VotesThisMonth,
		VotersAllTime:  totals.VotersAllTime,
		Approvals:      totals.Approvals,
		Denials:        totals.Denials,
		Months:         make([]types.StatsHistoryMonth, 0, len(rows)),
		GeneratedAt:    now,
	}

	votesFrom, reviewsFrom, usersFrom := "9999-99", "9999-99", "9999-99"

	if totals.UsersSince.Valid {
		first := totals.UsersSince.Time.UTC()
		usersFrom = time.Date(first.Year(), first.Month()+1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	}

	if totals.VotesSince.Valid {
		since := totals.VotesSince.Time
		out.VotesSince = &since
		votesFrom = since.UTC().Format("2006-01")
	}

	if totals.ReviewsSince.Valid {
		since := totals.ReviewsSince.Time
		out.ReviewsSince = &since
		reviewsFrom = since.UTC().Format("2006-01")
	}

	for _, r := range rows {
		m := types.StatsHistoryMonth{
			Month:      r.Month,
			NewBots:    r.NewBots,
			NewServers: r.NewServers,
		}

		if r.Month >= usersFrom {
			m.NewUsers = &r.NewUsers
		}

		if r.Month >= votesFrom {
			m.Votes, m.Voters = &r.Votes, &r.Voters
		}

		if r.Month >= reviewsFrom {
			m.Approved, m.Denied = &r.Approved, &r.Denied
		}

		out.Months = append(out.Months, m)
	}

	return out, nil
}

func Route(d uapi.RouteData, r *http.Request) uapi.HttpResponse {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cached != nil && time.Since(cached.GeneratedAt) < cacheFor {
		return uapi.HttpResponse{Json: cached}
	}

	out, err := Build(d.Context, time.Now())

	if err != nil {
		return resp.Err("Failed to fetch stats history", err)
	}

	cached = out

	return uapi.HttpResponse{Json: out}
}
