// Copyright (C) 2026 NodeByte LTD

package votes

import (
	"context"
	"testing"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestResetScheduleBookkeeping(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	testdb.Exec(t, "DELETE FROM automated_vote_resets")

	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	closing := ClosingPeriod(now)

	if done, err := q.AutomaticVoteResetDone(ctx, Date(closing)); err != nil || done {
		t.Fatalf("done before any reset = %v, %v", done, err)
	}

	if err := q.InsertAutomatedVoteReset(ctx, Date(closing)); err != nil {
		t.Fatal(err)
	}

	if done, err := q.AutomaticVoteResetDone(ctx, Date(closing)); err != nil || !done {
		t.Fatalf("done after recording = %v, %v", done, err)
	}

	period, counted := ManualResetPeriod(now)

	if !counted {
		t.Fatal("28 November should count for November")
	}

	for _, targetType := range []string{"bot", "server"} {
		if err := q.InsertManualVoteReset(ctx, db.InsertManualVoteResetParams{
			Period:      Date(period),
			TargetType:  pgtype.Text{String: targetType, Valid: true},
			TriggeredBy: pgtype.Text{String: "staff", Valid: true},
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := q.InsertManualVoteReset(ctx, db.InsertManualVoteResetParams{
		TargetType:  pgtype.Text{String: "team", Valid: true},
		TriggeredBy: pgtype.Text{String: "staff", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	manual, err := q.ListManualVoteResetTypes(ctx, Date(period))

	if err != nil || len(manual) != 2 {
		t.Fatalf("manual types for November = %v, %v, want bot and server only", manual, err)
	}

	latest, err := q.GetLatestAutomatedVoteReset(ctx)

	if err != nil || latest.Time.IsZero() {
		t.Fatalf("latest automatic reset = %v, %v", latest, err)
	}
}
