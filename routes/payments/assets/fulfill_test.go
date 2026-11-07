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

type fulfilled struct {
	provider, externalID string
	hours                int
}

func captureFulfillments(t *testing.T) *[]fulfilled {
	t.Helper()

	var calls []fulfilled
	prev := afterFulfill

	afterFulfill = func(_ context.Context, provider, externalID string, _ PerkData, hours int) {
		calls = append(calls, fulfilled{provider, externalID, hours})
	}

	t.Cleanup(func() { afterFulfill = prev })

	return &calls
}

func seedFreeBot(t *testing.T) (userID, botID string) {
	t.Helper()

	userID, botID = seedBot(t, time.Now().Add(-48*time.Hour), 1)
	testdb.Exec(t, "UPDATE bots SET premium = false WHERE bot_id = $1", botID)

	return userID, botID
}

func fulfillmentCount(t *testing.T, externalID string) int {
	t.Helper()

	var n int

	if err := state.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM payment_fulfillments WHERE external_id = $1", externalID).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func bronze(userID, botID string) PerkData {
	return PerkData{UserID: userID, ProductID: "premium", ProductName: "bronze", For: botID, ForType: "bot"}
}

func TestFulfillGrantsPremiumOnce(t *testing.T) {
	testdb.Pool(t)
	calls := captureFulfillments(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	session := "cs_test_" + testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE external_id = $1", session)

	ok, err := Fulfill(ctx, "stripe", session, bronze(userID, botID))

	if err != nil || !ok {
		t.Fatalf("first fulfill = %v, %v; want true, nil", ok, err)
	}

	if !premiumOf(t, botID) {
		t.Fatal("bot isn't premium after fulfilling")
	}

	if got := premiumHoursOf(t, botID); got != 720 {
		t.Fatalf("premium length = %vh, want 720h", got)
	}

	ok, err = Fulfill(ctx, "stripe", session, bronze(userID, botID))

	if err != nil || ok {
		t.Fatalf("repeat fulfill = %v, %v; want false, nil", ok, err)
	}

	if got := premiumHoursOf(t, botID); got != 720 {
		t.Fatalf("a repeated webhook changed premium to %vh", got)
	}

	if n := fulfillmentCount(t, session); n != 1 {
		t.Fatalf("fulfillment rows = %d, want 1", n)
	}

	if len(*calls) != 1 || (*calls)[0] != (fulfilled{"stripe", session, 720}) {
		t.Fatalf("after-fulfil hook calls = %+v, want one for the session", *calls)
	}
}

func TestFulfillRequiresAnID(t *testing.T) {
	testdb.Pool(t)
	captureFulfillments(t)
	userID, botID := seedFreeBot(t)

	if _, err := Fulfill(context.Background(), "stripe", "", bronze(userID, botID)); err == nil {
		t.Fatal("fulfilled without an external ID")
	}

	if premiumOf(t, botID) {
		t.Fatal("premium was granted without an external ID")
	}
}

func TestFulfillRejectsInvalidPurchases(t *testing.T) {
	testdb.Pool(t)
	captureFulfillments(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	_, premiumBot := seedBot(t, time.Now(), 720)
	_, pendingBot := seedFreeBot(t)
	testdb.Exec(t, "UPDATE bots SET type = 'pending' WHERE bot_id = $1", pendingBot)

	cases := map[string]PerkData{
		"unknown plan":     {UserID: userID, ProductID: "premium", ProductName: "platinum", For: botID, ForType: "bot"},
		"unknown product":  {UserID: userID, ProductID: "lootbox", ProductName: "bronze", For: botID, ForType: "bot"},
		"unknown listing":  bronze(userID, testdb.ID(t)),
		"already premium":  bronze(userID, premiumBot),
		"not approved":     bronze(userID, pendingBot),
		"bad listing type": {UserID: userID, ProductID: "premium", ProductName: "bronze", For: botID, ForType: "team"},
		"no buyer":         bronze("", botID),
	}

	for name, perk := range cases {
		t.Run(name, func(t *testing.T) {
			session := "cs_test_" + testdb.ID(t)
			testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE external_id = $1", session)

			ok, err := Fulfill(ctx, "stripe", session, perk)

			if ok || !errors.Is(err, ErrInvalidPurchase) {
				t.Fatalf("fulfill = %v, %v; want false, ErrInvalidPurchase", ok, err)
			}

			if n := fulfillmentCount(t, session); n != 0 {
				t.Fatalf("an invalid purchase left %d fulfillment rows", n)
			}
		})
	}

	if premiumOf(t, botID) || premiumOf(t, pendingBot) {
		t.Fatal("an invalid purchase granted premium")
	}
}

func TestFulfillSettlesCreditHold(t *testing.T) {
	testdb.Pool(t)
	captureFulfillments(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	session := "cs_test_" + testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE external_id = $1", session)
	testdb.Cleanup(t, "DELETE FROM payment_credit_holds WHERE external_id = $1", session)
	testdb.Cleanup(t, "DELETE FROM entity_vote_redeem_logs WHERE target_id = $1", botID)

	testdb.Exec(t, "INSERT INTO entity_vote_redeem_logs (target_id, target_type, credits) VALUES ($1, 'bot', 100)", botID)

	if err := HoldCredits(ctx, "stripe", session, userID, "bot", botID, 60, 30); err != nil {
		t.Fatal(err)
	}

	if ok, err := Fulfill(ctx, "stripe", session, bronze(userID, botID)); err != nil || !ok {
		t.Fatalf("fulfill = %v, %v", ok, err)
	}

	holdState, err := db.New(state.Pool).GetPaymentCreditHoldState(ctx, db.GetPaymentCreditHoldStateParams{Provider: "stripe", ExternalID: session})

	if err != nil {
		t.Fatal(err)
	}

	if holdState != "spent" {
		t.Fatalf("credit hold state = %q, want spent", holdState)
	}
}

func TestHoldCreditsSpendsOldestBatchesFirst(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	userID, botID := seedFreeBot(t)
	testdb.Cleanup(t, "DELETE FROM entity_vote_redeem_logs WHERE target_id = $1", botID)

	testdb.Exec(t, `INSERT INTO entity_vote_redeem_logs (target_id, target_type, credits, created_at) VALUES
		($1, 'bot', 30, NOW() - INTERVAL '2 days'),
		($1, 'bot', 50, NOW() - INTERVAL '1 day')`, botID)

	session := "cs_test_" + testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM payment_credit_holds WHERE external_id = $1", session)

	if err := HoldCredits(ctx, "stripe", session, userID, "bot", botID, 40, 20); err != nil {
		t.Fatal(err)
	}

	rows, err := state.Pool.Query(ctx, "SELECT redeemed_credits FROM entity_vote_redeem_logs WHERE target_id = $1 ORDER BY created_at", botID)

	if err != nil {
		t.Fatal(err)
	}

	var spent []int

	for rows.Next() {
		var n int

		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}

		spent = append(spent, n)
	}

	rows.Close()

	if len(spent) != 2 || spent[0] != 30 || spent[1] != 10 {
		t.Fatalf("redeemed per batch = %v, want [30 10]", spent)
	}

	tooMuch := "cs_test_" + testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM payment_credit_holds WHERE external_id = $1", tooMuch)

	if err := HoldCredits(ctx, "stripe", tooMuch, userID, "bot", botID, 41, 20); !errors.Is(err, ErrNotEnoughCredits) {
		t.Fatalf("holding more than is left = %v, want ErrNotEnoughCredits", err)
	}

	if got, err := AvailableCredits(ctx, "bot", botID); err != nil || got != 40 {
		t.Fatalf("available after a failed hold = %d, %v; want 40", got, err)
	}
}

func TestRecordGift(t *testing.T) {
	testdb.Pool(t)
	ctx := context.Background()
	q := db.New(state.Pool)
	gifter := testdb.ID(t)
	gift, notGift := "cs_test_"+testdb.ID(t), "cs_test_"+testdb.ID(t)
	testdb.Cleanup(t, "DELETE FROM payment_gifts WHERE external_id = ANY($1)", []string{gift, notGift})

	RecordGift(ctx, "stripe", notGift, gifter, CreatePerkData{GiftMessage: "ignored"})

	if _, err := q.GetPaymentGift(ctx, db.GetPaymentGiftParams{Provider: "stripe", ExternalID: notGift}); err == nil {
		t.Fatal("a normal purchase was recorded as a gift")
	}

	RecordGift(ctx, "stripe", gift, gifter, CreatePerkData{Gift: true, GiftMessage: "  enjoy  ", GiftAnonymous: true})
	RecordGift(ctx, "stripe", gift, testdb.ID(t), CreatePerkData{Gift: true, GiftMessage: "second"})

	row, err := q.GetPaymentGift(ctx, db.GetPaymentGiftParams{Provider: "stripe", ExternalID: gift})

	if err != nil {
		t.Fatal(err)
	}

	if row.GifterID != gifter || row.Message != "enjoy" || !row.Anonymous {
		t.Fatalf("gift = %+v, want the first record with a trimmed message", row)
	}
}
