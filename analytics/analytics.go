// Copyright (C) 2026 NodeByte LTD

package analytics

import (
	"context"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/types"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	MinDays     = 7
	MaxDays     = 365
	DefaultDays = 30
)

func Snapshot(ctx context.Context) (int64, error) {
	q := db.New(state.Pool)

	bots, err := q.SnapshotBotStats(ctx)

	if err != nil {
		return 0, err
	}

	servers, err := q.SnapshotServerStats(ctx)

	if err != nil {
		return 0, err
	}

	if _, err := q.PruneEntityStats(ctx); err != nil {
		return 0, err
	}

	return bots + servers, nil
}

func Get(ctx context.Context, targetType, targetID string, days int) (*types.EntityAnalytics, error) {
	days = max(MinDays, min(days, MaxDays))

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -(days - 1))

	q := db.New(state.Pool)

	snaps, err := q.GetEntityStatsSnapshots(ctx, db.GetEntityStatsSnapshotsParams{
		TargetType: targetType,
		TargetID:   targetID,
		Since:      pgtype.Date{Time: start.AddDate(0, 0, -1), Valid: true},
	})

	if err != nil {
		return nil, err
	}

	votes, err := q.GetEntityDailyVotes(ctx, db.GetEntityDailyVotesParams{
		TargetType: targetType,
		TargetID:   targetID,
		Since:      pgtype.Date{Time: start, Valid: true},
	})

	if err != nil {
		return nil, err
	}

	snapByDay := make(map[string]db.GetEntityStatsSnapshotsRow, len(snaps))
	for _, s := range snaps {
		snapByDay[dayKey(s.Day.Time)] = s
	}

	voteByDay := make(map[string]db.GetEntityDailyVotesRow, len(votes))
	for _, v := range votes {
		voteByDay[dayKey(v.Day.Time)] = v
	}

	out := &types.EntityAnalytics{Days: make([]types.EntityAnalyticsDay, 0, days)}

	for _, s := range snaps {
		if !s.Day.Time.Before(start) {
			d := dayKey(s.Day.Time)
			out.TrackedSince = &d
			break
		}
	}

	for i := range days {
		t := start.AddDate(0, 0, i)
		key := dayKey(t)

		day := types.EntityAnalyticsDay{Day: key}

		if v, ok := voteByDay[key]; ok {
			day.Votes = int(v.Votes)
			day.AutoVotes = int(v.AutoVotes)
		}

		if s, ok := snapByDay[key]; ok {
			if prev, ok := snapByDay[dayKey(t.AddDate(0, 0, -1))]; ok {
				day.Views = delta(s.Clicks, prev.Clicks)
				day.InviteClicks = delta(s.InviteClicks, prev.InviteClicks)
			}

			if targetType == "bot" {
				day.Servers = ptr(int(s.Servers))
				day.Users = ptr(int(s.Users))
			} else {
				day.TotalMembers = ptr(int(s.TotalMembers))
				day.OnlineMembers = ptr(int(s.OnlineMembers))
			}
		}

		out.Days = append(out.Days, day)
	}

	return out, nil
}

func dayKey(t time.Time) string {
	return t.UTC().Format(time.DateOnly)
}

func delta(cur, prev int32) *int {
	return ptr(max(0, int(cur-prev)))
}

func ptr(v int) *int {
	return &v
}
