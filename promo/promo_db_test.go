package promo

import (
	"context"
	"testing"
	"time"

	"popplio/testdb"
)

type seedCode struct {
	products         string
	percent          *int
	amountCents      *int
	maxRedemptions   *int
	expiresAt        *time.Time
	oncePerUser      bool
	newCustomersOnly bool
	active           bool
}

func insertCode(t *testing.T, c seedCode) string {
	t.Helper()

	code := "T" + testdb.ID(t)

	testdb.Exec(t, `INSERT INTO promo_codes (code, products, percent_off, amount_off_cents, duration, max_redemptions, expires_at, once_per_user, new_customers_only, active, stripe_coupon_id, stripe_promotion_code_id, created_by)
		VALUES ($1, string_to_array($2, ','), $3, $4, 'once', $5, $6, $7, $8, $9, 'co_test', 'promo_test', 'test')`,
		code, c.products, c.percent, c.amountCents, c.maxRedemptions, c.expiresAt, c.oncePerUser, c.newCustomersOnly, c.active)
	testdb.Cleanup(t, "DELETE FROM promo_codes WHERE code = $1", code)

	return code
}

func redeem(t *testing.T, code, userID string) {
	t.Helper()

	session := "cs_test_" + testdb.ID(t)

	testdb.Exec(t, `INSERT INTO promo_code_redemptions (promo_code_id, user_id, product, stripe_session_id, status, completed_at)
		SELECT id, $2, 'premium', $3, 'completed', NOW() FROM promo_codes WHERE code = $1`, code, userID, session)
}

func TestValidateAgainstDatabase(t *testing.T) {
	testdb.Pool(t)

	ctx := context.Background()
	user := testdb.ID(t)
	past := time.Now().Add(-time.Hour)

	rejects := []struct {
		name    string
		code    seedCode
		product string
		want    string
	}{
		{"inactive", seedCode{products: "premium", percent: intp(10)}, ProductPremium, "This promo code is no longer active"},
		{"expired", seedCode{products: "premium", percent: intp(10), expiresAt: &past, active: true}, ProductPremium, "This promo code has expired"},
		{"wrong product", seedCode{products: "autovote", percent: intp(10), active: true}, ProductPremium, "This promo code can't be used for premium"},
		{"under minimum", seedCode{products: "premium", amountCents: intp(160), active: true}, ProductPremium, "With this promo code the total would be under the $0.50 minimum card payment. Try using fewer credits"},
	}

	for _, c := range rejects {
		code := insertCode(t, c.code)
		_, _, err := Validate(ctx, user, code, c.product, 199)
		rejected(t, err, c.want)
	}

	_, _, err := Validate(ctx, user, "TNOPE"+testdb.ID(t), ProductPremium, 199)
	rejected(t, err, "This promo code doesn't exist")

	full := insertCode(t, seedCode{products: "premium", percent: intp(10), maxRedemptions: intp(1), active: true})
	redeem(t, full, testdb.ID(t))
	_, _, err = Validate(ctx, user, full, ProductPremium, 199)
	rejected(t, err, "This promo code has been fully redeemed")

	once := insertCode(t, seedCode{products: "premium", percent: intp(10), oncePerUser: true, active: true})
	redeem(t, once, user)
	_, _, err = Validate(ctx, user, once, ProductPremium, 199)
	rejected(t, err, "You've already used this promo code")

	if _, _, err := Validate(ctx, testdb.ID(t), once, ProductPremium, 199); err != nil {
		t.Errorf("another user should still be able to use a once-per-user code: %v", err)
	}

	fresh := insertCode(t, seedCode{products: "premium,autovote", percent: intp(100), newCustomersOnly: true, active: true})
	paying := testdb.ID(t)
	testdb.Exec(t, `INSERT INTO payment_fulfillments (provider, external_id, user_id, product_id, for_type, for_id) VALUES ('stripe', $1, $2, 'premium', 'bot', '1')`, "cs_test_"+paying, paying)
	testdb.Cleanup(t, "DELETE FROM payment_fulfillments WHERE user_id = $1", paying)
	_, _, err = Validate(ctx, paying, fresh, ProductAutoVote, 299)
	rejected(t, err, "This promo code is only for new customers")

	_, preview, err := Validate(ctx, user, " "+fresh+" ", ProductAutoVote, 299)

	if err != nil {
		t.Fatalf("valid code rejected: %v", err)
	}

	if preview.DiscountCents != 299 || preview.TotalCents != 0 || preview.Code != fresh {
		t.Errorf("unexpected preview: %+v", preview)
	}
}
