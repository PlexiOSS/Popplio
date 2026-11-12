// Copyright (C) 2026 NodeByte LTD

package autovote

import (
	"context"
	"slices"
	"testing"

	"popplio/db"
	"popplio/testdb"
)

func TestIsShareSpike(t *testing.T) {
	cases := []struct {
		name                                       string
		autoDay, totalDay, autoBefore, totalBefore int
		want                                       bool
	}{
		{"too few automatic votes", 9, 9, 0, 50, false},
		{"mostly automatic, up from manual", 12, 15, 5, 100, true},
		{"new listing with only automatic votes", 10, 10, 0, 0, true},
		{"always mostly automatic", 30, 40, 150, 200, false},
		{"under half automatic", 10, 30, 0, 100, false},
	}

	for _, c := range cases {
		if got := IsShareSpike(c.autoDay, c.totalDay, c.autoBefore, c.totalBefore); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCardLimit(t *testing.T) {
	q := db.New(testdb.Pool(t))
	ctx := context.Background()
	card := "card_" + testdb.ID(t)

	var users []string

	for i := range 5 {
		user := testdb.ID(t)
		users = append(users, user)
		status := "active"

		if i == 4 {
			status = "canceled"
		}

		testdb.Exec(t, `INSERT INTO auto_vote_subscriptions (user_id, stripe_customer_id, stripe_subscription_id, status, card_fingerprint, created_at)
			VALUES ($1, 'cus_test', $2, $3, $4, NOW() - make_interval(hours => $5))`, user, "sub_"+user, status, card, 10-i)
		testdb.Cleanup(t, "DELETE FROM auto_vote_subscriptions WHERE user_id = $1", user)
	}

	over, err := q.ListAutoVoteUsersOverCardLimit(ctx, MaxAccountsPerCard)

	if err != nil {
		t.Fatal(err)
	}

	for i, user := range users {
		want := i == 3

		if got := slices.Contains(over, user); got != want {
			t.Errorf("account %d over the limit = %v, want %v", i+1, got, want)
		}
	}

	accounts, err := q.ListAutoVoteAccountsOnCard(ctx, pgText(card))

	if err != nil {
		t.Fatal(err)
	}

	if len(accounts) != 4 || accounts[0] != users[0] {
		t.Errorf("accounts on card = %v, want the 4 active ones oldest first", accounts)
	}
}

func TestShareSpikeQuery(t *testing.T) {
	q := db.New(testdb.Pool(t))
	ctx := context.Background()
	bot := testdb.ID(t)

	vote := func(auto bool, daysAgo int) {
		user := testdb.ID(t)
		testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", user, "tok-"+user)
		testdb.Exec(t, `INSERT INTO entity_votes (author, target_id, target_type, upvote, auto, created_at)
			VALUES ($1, $2, 'bot', true, $3, NOW() - make_interval(days => $4) - INTERVAL '1 hour')`, user, bot, auto, daysAgo)
		testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", user)
	}

	testdb.Cleanup(t, "DELETE FROM entity_votes WHERE target_id = $1", bot)

	for range 12 {
		vote(true, 0)
	}

	for range 2 {
		vote(false, 0)
	}

	for range 10 {
		vote(false, 3)
	}

	rows, err := q.ListAutoVoteShareSpikes(ctx, SpikeMinAutoVotes)

	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		if r.TargetID != bot {
			continue
		}

		if r.AutoDay != 12 || r.TotalDay != 14 || r.AutoBefore != 0 || r.TotalBefore != 10 || r.AutoVoters != 12 {
			t.Fatalf("counts = %+v", r)
		}

		if !IsShareSpike(int(r.AutoDay), int(r.TotalDay), int(r.AutoBefore), int(r.TotalBefore)) {
			t.Fatal("expected a spike")
		}

		return
	}

	t.Fatal("listing with a spike wasn't returned")
}
