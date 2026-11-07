// Copyright (C) 2026 NodeByte LTD

package assets

import (
	"context"
	"errors"
	"testing"
	"time"

	"popplio/db"
	"popplio/state"
	"popplio/testdb"
)

func TestApplyBoosterPremium(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	now := time.Now()
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE provider = 'booster' AND user_id = $1", userID)

	hours, err := ApplyBoosterPremium(ctx, userID, "bot", botID, now)

	if err != nil || hours != 720 {
		t.Fatalf("first apply = %d, %v; want 720, nil", hours, err)
	}

	if !premiumOf(t, botID) || premiumHoursOf(t, botID) != 720 {
		t.Fatalf("premium = %v, %vh; want true, 720h", premiumOf(t, botID), premiumHoursOf(t, botID))
	}

	claim, err := db.New(state.Pool).GetUserLastBoosterClaim(ctx, userID)

	if err != nil || !claim.Valid {
		t.Fatalf("last claim = %v, %v; want set", claim, err)
	}

	if hours, err := ApplyBoosterPremium(ctx, userID, "bot", botID, now); err != nil || hours != 0 {
		t.Fatalf("same-day apply = %d, %v; want 0, nil", hours, err)
	}

	if got := premiumHoursOf(t, botID); got != 720 {
		t.Fatalf("a repeat on the same day changed premium to %vh", got)
	}

	if hours, err := ApplyBoosterPremium(ctx, userID, "bot", botID, now.Add(30*24*time.Hour)); err != nil || hours != 720 {
		t.Fatalf("next month = %d, %v; want 720, nil", hours, err)
	}

	if got := premiumHoursOf(t, botID); got != 1440 {
		t.Fatalf("a renewal on active premium should extend it to 1440h, got %vh", got)
	}
}

func TestApplyBoosterPremiumRejectsBadListings(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	testdb.Exec(t, "UPDATE bots SET type = 'denied' WHERE bot_id = $1", botID)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE provider = 'booster' AND user_id = $1", userID)

	for name, c := range map[string][2]string{
		"not approved": {"bot", botID},
		"missing":      {"server", testdb.ID(t)},
		"bad type":     {"team", botID},
	} {
		t.Run(name, func(t *testing.T) {
			var targetErr *TargetError

			if _, err := ApplyBoosterPremium(ctx, userID, c[0], c[1], time.Now()); !errors.As(err, &targetErr) {
				t.Fatalf("err = %v, want a TargetError", err)
			}
		})
	}

	if premiumOf(t, botID) {
		t.Fatal("a denied bot got booster premium")
	}

	if claim, _ := db.New(state.Pool).GetUserLastBoosterClaim(ctx, userID); claim.Valid {
		t.Fatal("a failed apply used up the month")
	}
}

func TestBoosterSyncQueries(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	a, b := testdb.ID(t), testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM booster_perks WHERE user_id = ANY($1)", []string{a, b})

	started, err := q.StartBoosting(ctx, []string{a, b})

	if err != nil || len(started) != 2 {
		t.Fatalf("start = %v, %v; want both", started, err)
	}

	if again, err := q.StartBoosting(ctx, []string{a, b}); err != nil || len(again) != 0 {
		t.Fatalf("starting again = %v, %v; want none", again, err)
	}

	stopped, err := q.StopBoosting(ctx, []string{a})

	if err != nil {
		t.Fatal(err)
	}

	var gotB bool

	for _, id := range stopped {
		if id == a {
			t.Fatal("a booster still in the list was stopped")
		}

		gotB = gotB || id == b
	}

	if !gotB {
		t.Fatalf("stopped = %v, want it to include the missing booster", stopped)
	}

	if boosting, _ := q.IsBoosting(ctx, b); boosting {
		t.Fatal("a stopped booster is still boosting")
	}

	if restarted, err := q.StartBoosting(ctx, []string{a, b}); err != nil || len(restarted) != 1 || restarted[0] != b {
		t.Fatalf("restart = %v, %v; want only the returning booster", restarted, err)
	}
}

func TestDueBoosterRenewals(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	userID, botID := seedFreeBot(t)
	testdb.Cleanup(t, "DELETE FROM booster_perks WHERE user_id = $1", userID)

	isDue := func() bool {
		rows, err := q.ListDueBoosterRenewals(ctx)

		if err != nil {
			t.Fatal(err)
		}

		for _, r := range rows {
			if r.UserID == userID {
				return true
			}
		}

		return false
	}

	testdb.Exec(t, "INSERT INTO booster_perks (user_id, boosting, boosting_since) VALUES ($1, true, NOW())", userID)

	if isDue() {
		t.Fatal("due without a listing picked")
	}

	testdb.Exec(t, "UPDATE booster_perks SET target_type = 'bot', target_id = $2 WHERE user_id = $1", userID, botID)

	if !isDue() {
		t.Fatal("not due with a listing and no claim yet")
	}

	testdb.Exec(t, "UPDATE users SET last_booster_claim = NOW() - INTERVAL '29 days' WHERE user_id = $1", userID)

	if isDue() {
		t.Fatal("due before 30 days")
	}

	testdb.Exec(t, "UPDATE users SET last_booster_claim = NOW() - INTERVAL '31 days' WHERE user_id = $1", userID)

	if !isDue() {
		t.Fatal("not due after 30 days")
	}

	testdb.Exec(t, "UPDATE booster_perks SET boosting = false WHERE user_id = $1", userID)

	if isDue() {
		t.Fatal("due after the boost ended")
	}
}
