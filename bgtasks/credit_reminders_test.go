// Copyright (C) 2026 NodeByte LTD

package bgtasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"

	"github.com/jackc/pgx/v5"
)

func TestCreditReminderMessage(t *testing.T) {
	reset := time.Date(2026, 12, 3, 4, 0, 0, 0, time.UTC)

	cases := []struct {
		redeem, credits int
		want            string
	}{
		{0, 0, ""},
		{1, 0, "Votes reset on 3 December. 1 of your listings has votes you can still turn into credits."},
		{3, 0, "Votes reset on 3 December. 3 of your listings have votes you can still turn into credits."},
		{0, 1, "1 of your listings has credits waiting to be spent in the Credit shop."},
		{2, 1, "Votes reset on 3 December. 2 of your listings have votes you can still turn into credits, and 1 has credits waiting to be spent."},
		{1, 4, "Votes reset on 3 December. 1 of your listings has votes you can still turn into credits, and 4 have credits waiting to be spent."},
	}

	for _, c := range cases {
		if got := CreditReminderMessage(reset, c.redeem, c.credits); got != c.want {
			t.Errorf("(%d, %d) = %q\nwant %q", c.redeem, c.credits, got, c.want)
		}
	}
}

func TestCreditReminderQueries(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	voter, bot := testdb.ID(t), testdb.ID(t)
	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", voter, "tok-"+voter)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", voter)
	testdb.Cleanup(t, "DELETE FROM entity_votes WHERE target_id = $1", bot)
	testdb.Cleanup(t, "DELETE FROM entity_vote_redeem_logs WHERE target_id = $1", bot)
	testdb.Cleanup(t, "DELETE FROM credit_reminders WHERE user_id = $1", voter)

	for i := 0; i < 12; i++ {
		testdb.Exec(t, "INSERT INTO entity_votes (target_id, target_type, author, upvote) VALUES ($1, 'bot', $2, true)", bot, voter)
	}

	testdb.Exec(t, "INSERT INTO entity_votes (target_id, target_type, author, upvote, auto) VALUES ($1, 'bot', $2, true, true)", bot, voter)
	testdb.Exec(t, "INSERT INTO entity_votes (target_id, target_type, author, upvote, void) VALUES ($1, 'bot', $2, true, true)", bot, voter)

	votesFor := func(min int32) int32 {
		rows, err := q.ListRedeemableVoteTotals(ctx, min)

		if err != nil {
			t.Fatal(err)
		}

		for _, r := range rows {
			if r.TargetID == bot {
				return r.Votes
			}
		}

		return -1
	}

	if got := votesFor(10); got != 12 {
		t.Fatalf("redeemable votes = %d, want 12 (auto and voided votes don't count)", got)
	}

	if got := votesFor(13); got != -1 {
		t.Fatalf("a listing under the minimum was included with %d votes", got)
	}

	testdb.Exec(t, "INSERT INTO entity_vote_redeem_logs (target_id, target_type, credits, redeemed_credits) VALUES ($1, 'bot', 100, 70)", bot)

	rows, err := q.ListUnspentCreditTotals(ctx, 30)

	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, r := range rows {
		if r.TargetID == bot {
			found = r.Credits == 30
		}
	}

	if !found {
		t.Fatal("30 unspent credits weren't listed at a minimum of 30")
	}

	if _, err := q.MarkCreditReminderSent(ctx, db.MarkCreditReminderSentParams{UserID: voter, Period: "2099-01-01"}); err != nil {
		t.Fatal(err)
	}

	if _, err := q.MarkCreditReminderSent(ctx, db.MarkCreditReminderSentParams{UserID: voter, Period: "2099-01-01"}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("second reminder in the same period = %v, want no row", err)
	}
}
