// Copyright (C) 2026 NodeByte LTD

package votes

import (
	"context"
	"slices"
	"time"

	"popplio/db"
	"popplio/state"

	"github.com/jackc/pgx/v5/pgtype"
)

const ManualResetWindow = 7 * 24 * time.Hour

var ResetTargetTypes = []string{"bot", "server", "team"}

func MonthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func ClosingPeriod(now time.Time) time.Time {
	return MonthStart(now).AddDate(0, -1, 0)
}

func ManualResetPeriod(now time.Time) (time.Time, bool) {
	next := MonthStart(now).AddDate(0, 1, 0)

	if next.Sub(now.UTC()) <= ManualResetWindow {
		return MonthStart(now), true
	}

	return time.Time{}, false
}

func Date(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func NextReset(ctx context.Context, now time.Time) (time.Time, error) {
	current := MonthStart(now)
	next := current.AddDate(0, 1, 0)

	manual, err := db.New(state.Pool).ListManualVoteResetTypes(ctx, Date(current))

	if err != nil {
		return next, err
	}

	for _, targetType := range ResetTargetTypes {
		if !slices.Contains(manual, targetType) {
			return next, nil
		}
	}

	return next.AddDate(0, 1, 0), nil
}
