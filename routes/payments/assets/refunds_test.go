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

func seedBot(t *testing.T, premiumStart time.Time, premiumHours int) (userID, botID string) {
	t.Helper()

	userID, botID = testdb.ID(t), testdb.ID(t)

	testdb.Exec(t, "INSERT INTO users (user_id, api_token) VALUES ($1, $2)", userID, "tok-"+userID)
	testdb.Cleanup(t, "DELETE FROM users WHERE user_id = $1", userID)

	var vanity string

	if err := state.Pool.QueryRow(context.Background(),
		"INSERT INTO vanity (target_id, target_type, code) VALUES ($1, 'bot', $2) RETURNING itag::text", botID, "b"+botID,
	).Scan(&vanity); err != nil {
		t.Fatalf("vanity: %v", err)
	}

	testdb.Cleanup(t, "DELETE FROM vanity WHERE itag = $1::uuid", vanity)

	testdb.Exec(t, `INSERT INTO bots (bot_id, client_id, tags, prefix, short, long, extra_links, invite, vanity_ref, owner, type, premium, start_premium_period, premium_period_length)
		VALUES ($1, $1, '{}', '!', 'short', 'long', '[]', 'https://example.com', $2::uuid, $3, 'approved', true, $4, make_interval(hours => $5))`,
		botID, vanity, userID, premiumStart, premiumHours)
	testdb.Cleanup(t, "DELETE FROM bots WHERE bot_id = $1", botID)

	return userID, botID
}

func seedPurchase(t *testing.T, sessionID, userID, botID string, fulfilledAt time.Time, hours *int, credits int) {
	t.Helper()

	testdb.Exec(t, `INSERT INTO payment_fulfillments (provider, external_id, user_id, product_id, for_type, for_id, fulfilled_at, premium_hours)
		VALUES ('stripe', $1, $2, 'premium', 'bot', $3, $4, $5)`, sessionID, userID, botID, fulfilledAt, hours)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE external_id = $1", sessionID)

	if credits > 0 {
		testdb.Exec(t, `INSERT INTO payment_credit_holds (provider, external_id, user_id, target_type, target_id, credits, discount_cents, state)
			VALUES ('stripe', $1, $2, 'bot', $3, $4, $5, 'spent')`, sessionID, userID, botID, credits, credits/2)
		testdb.Cleanup(t, "DELETE FROM payment_credit_holds WHERE external_id = $1", sessionID)
		testdb.Cleanup(t, "DELETE FROM entity_vote_redeem_logs WHERE target_id = $1", botID)
	}
}

func premiumOf(t *testing.T, botID string) bool {
	t.Helper()

	var premium bool

	if err := state.Pool.QueryRow(context.Background(), "SELECT premium FROM bots WHERE bot_id = $1", botID).Scan(&premium); err != nil {
		t.Fatal(err)
	}

	return premium
}

func premiumHoursOf(t *testing.T, botID string) float64 {
	t.Helper()

	var hours float64

	if err := state.Pool.QueryRow(context.Background(), "SELECT EXTRACT(EPOCH FROM premium_period_length)::float8 / 3600 FROM bots WHERE bot_id = $1", botID).Scan(&hours); err != nil {
		t.Fatal(err)
	}

	return hours
}

func hoursPtr(h int) *int { return &h }

func returnedCredits(t *testing.T, botID string) int {
	t.Helper()

	var credits int

	if err := state.Pool.QueryRow(context.Background(), "SELECT COALESCE(SUM(credits), 0)::int FROM entity_vote_redeem_logs WHERE target_id = $1", botID).Scan(&credits); err != nil {
		t.Fatal(err)
	}

	return credits
}

func TestRevokeRefundRemovesPremiumAndReturnsCredits(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	session := "cs_test_" + botID
	seedPurchase(t, session, userID, botID, granted, hoursPtr(720), 100)

	res, err := revoke(context.Background(), "stripe", session, RevokeRefunded, true)

	if err != nil {
		t.Fatal(err)
	}

	if !res.PremiumRemoved || !res.PremiumEnded || res.CreditsReturned != 100 || res.AlreadyRevoked {
		t.Fatalf("unexpected result: %+v", res)
	}

	if premiumOf(t, botID) {
		t.Error("bot still premium after refund")
	}

	if got := returnedCredits(t, botID); got != 100 {
		t.Errorf("returned credits = %d, want 100", got)
	}

	again, err := revoke(context.Background(), "stripe", session, RevokeRefunded, true)

	if err != nil || !again.AlreadyRevoked {
		t.Fatalf("second revoke should be a no-op: %+v %v", again, err)
	}

	if got := returnedCredits(t, botID); got != 100 {
		t.Errorf("credits returned twice: %d", got)
	}
}

func TestRevokeDisputeKeepsCredits(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	session := "cs_test_" + botID
	seedPurchase(t, session, userID, botID, granted, hoursPtr(720), 60)

	res, err := revoke(context.Background(), "stripe", session, RevokeDisputed, false)

	if err != nil || !res.PremiumRemoved || res.CreditsReturned != 0 {
		t.Fatalf("unexpected result: %+v %v", res, err)
	}

	if got := returnedCredits(t, botID); got != 0 {
		t.Errorf("dispute returned %d credits", got)
	}
}

func TestRevokeLeavesNewerPremiumAlone(t *testing.T) {
	testdb.Pool(t)

	purchased := time.Now().Add(-48 * time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, time.Now().Add(-time.Hour), 720)
	session := "cs_test_" + botID
	seedPurchase(t, session, userID, botID, purchased, hoursPtr(720), 0)

	res, err := revoke(context.Background(), "stripe", session, RevokeRefunded, true)

	if err != nil || res.PremiumRemoved {
		t.Fatalf("premium from a later grant was removed: %+v %v", res, err)
	}

	if !premiumOf(t, botID) {
		t.Error("bot lost premium it got from somewhere else")
	}
}

func TestRevokeUnknownPurchase(t *testing.T) {
	testdb.Pool(t)

	if _, err := revoke(context.Background(), "stripe", "cs_test_missing_"+testdb.ID(t), RevokeRefunded, true); !errors.Is(err, ErrNotFulfilled) {
		t.Fatalf("err = %v, want ErrNotFulfilled", err)
	}
}

func TestRevokeKeepsTimeAddedLater(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720+168)
	session := "cs_test_" + botID
	seedPurchase(t, session, userID, botID, granted, hoursPtr(720), 0)

	res, err := revoke(context.Background(), "stripe", session, RevokeRefunded, true)

	if err != nil || !res.PremiumRemoved || res.PremiumEnded || res.HoursRemoved != 720 {
		t.Fatalf("unexpected result: %+v %v", res, err)
	}

	if !premiumOf(t, botID) {
		t.Error("bot lost the premium days added after the refunded purchase")
	}

	if got := premiumHoursOf(t, botID); got != 168 {
		t.Errorf("premium length = %vh, want 168h", got)
	}
}

func TestRevokeWithoutStoredHoursRemovesAll(t *testing.T) {
	testdb.Pool(t)

	granted := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	userID, botID := seedBot(t, granted, 720)
	session := "cs_test_" + botID
	seedPurchase(t, session, userID, botID, granted, nil, 0)

	res, err := revoke(context.Background(), "stripe", session, RevokeRefunded, true)

	if err != nil || !res.PremiumRemoved || !res.PremiumEnded {
		t.Fatalf("unexpected result: %+v %v", res, err)
	}

	if premiumOf(t, botID) {
		t.Error("bot still premium")
	}
}

func TestApplyPremiumDaysExtendsActivePremium(t *testing.T) {
	testdb.Pool(t)

	start := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	_, botID := seedBot(t, start, 720)
	q := db.New(state.Pool)

	if err := q.ApplyBotPremiumDays(context.Background(), db.ApplyBotPremiumDaysParams{Hours: 168, BotID: botID}); err != nil {
		t.Fatal(err)
	}

	var gotStart time.Time

	if err := state.Pool.QueryRow(context.Background(), "SELECT start_premium_period FROM bots WHERE bot_id = $1", botID).Scan(&gotStart); err != nil {
		t.Fatal(err)
	}

	if !gotStart.Equal(start) {
		t.Errorf("start moved from %v to %v", start, gotStart)
	}

	if got := premiumHoursOf(t, botID); got != 888 {
		t.Errorf("premium length = %vh, want 888h", got)
	}

	testdb.Exec(t, "UPDATE bots SET start_premium_period = NOW() - interval '1000 hours' WHERE bot_id = $1", botID)

	if err := q.ApplyBotPremiumDays(context.Background(), db.ApplyBotPremiumDaysParams{Hours: 24, BotID: botID}); err != nil {
		t.Fatal(err)
	}

	if got := premiumHoursOf(t, botID); got != 24 {
		t.Errorf("expired premium should restart at 24h, got %vh", got)
	}
}
