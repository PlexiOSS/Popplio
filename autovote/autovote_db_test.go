// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"context"
	"testing"

	"popplio/db"
	"popplio/testdb"
)

func seedSubscriber(t *testing.T) string {
	t.Helper()

	user := testdb.ID(t)

	testdb.Exec(t, `INSERT INTO auto_vote_subscriptions (user_id, stripe_customer_id, stripe_subscription_id, status) VALUES ($1, 'cus_test', $2, 'active')`, user, "sub_test_"+user)
	testdb.Cleanup(t, "DELETE FROM auto_vote_subscriptions WHERE user_id = $1", user)
	testdb.Cleanup(t, "DELETE FROM auto_vote_targets WHERE user_id = $1", user)
	testdb.Cleanup(t, "DELETE FROM auto_vote_settings WHERE user_id = $1", user)

	return user
}

func TestPausedTargetsAreNotDue(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(testdb.Pool(t))
	user := seedSubscriber(t)

	testdb.Exec(t, `INSERT INTO auto_vote_targets (user_id, target_type, target_id, paused) VALUES ($1, 'bot', '1', false), ($1, 'bot', '2', true)`, user)

	if err := q.SetAutoVoteNotify(ctx, db.SetAutoVoteNotifyParams{UserID: user, Notify: "each"}); err != nil {
		t.Fatal(err)
	}

	due, err := q.ListDueAutoVoteTargets(ctx)

	if err != nil {
		t.Fatal(err)
	}

	var mine []db.ListDueAutoVoteTargetsRow

	for _, row := range due {
		if row.UserID == user {
			mine = append(mine, row)
		}
	}

	if len(mine) != 1 || mine[0].TargetID != "1" {
		t.Fatalf("due targets = %+v, want only the unpaused one", mine)
	}

	if mine[0].Notify != "each" {
		t.Fatalf("notify = %q, want each", mine[0].Notify)
	}
}

func TestDailySummaryWaitsADay(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(testdb.Pool(t))
	user := seedSubscriber(t)

	isDue := func() bool {
		rows, err := q.ListDueAutoVoteSummaries(ctx)

		if err != nil {
			t.Fatal(err)
		}

		for _, row := range rows {
			if row.UserID == user {
				return true
			}
		}

		return false
	}

	if err := q.SetAutoVoteNotify(ctx, db.SetAutoVoteNotifyParams{UserID: user, Notify: "daily"}); err != nil {
		t.Fatal(err)
	}

	if isDue() {
		t.Fatal("a summary was due straight after turning daily summaries on")
	}

	testdb.Exec(t, `UPDATE auto_vote_settings SET last_summary_at = NOW() - INTERVAL '25 hours' WHERE user_id = $1`, user)

	if !isDue() {
		t.Fatal("a summary wasn't due after a day")
	}

	if err := q.SetAutoVoteNotify(ctx, db.SetAutoVoteNotifyParams{UserID: user, Notify: "daily"}); err != nil {
		t.Fatal(err)
	}

	if !isDue() {
		t.Fatal("saving the same setting again reset the summary clock")
	}

	if err := q.MarkAutoVoteSummarySent(ctx, user); err != nil {
		t.Fatal(err)
	}

	if isDue() {
		t.Fatal("a summary was still due after one was sent")
	}

	if err := q.SetAutoVoteNotify(ctx, db.SetAutoVoteNotifyParams{UserID: user, Notify: "off"}); err != nil {
		t.Fatal(err)
	}

	testdb.Exec(t, `UPDATE auto_vote_settings SET last_summary_at = NOW() - INTERVAL '25 hours' WHERE user_id = $1`, user)

	if isDue() {
		t.Fatal("a summary was due with summaries turned off")
	}
}
