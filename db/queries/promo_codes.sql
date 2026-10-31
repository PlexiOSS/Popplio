-- name: InsertPromoCode :one
INSERT INTO promo_codes (code, description, products, percent_off, amount_off_cents, duration, duration_months, max_redemptions, expires_at, once_per_user, new_customers_only, stripe_coupon_id, stripe_promotion_code_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id;

-- name: ListPromoCodes :many
SELECT p.id, p.code, p.description, p.products, p.percent_off, p.amount_off_cents, p.duration, p.duration_months, p.max_redemptions, p.expires_at, p.once_per_user, p.new_customers_only, p.active, p.stripe_coupon_id, p.stripe_promotion_code_id, p.created_by, p.created_at, p.updated_at,
    (SELECT count(*) FROM promo_code_redemptions r WHERE r.promo_code_id = p.id AND r.status = 'completed')::int AS redemptions
FROM promo_codes p
ORDER BY p.created_at DESC;

-- name: GetPromoCodeByCode :one
SELECT id, code, description, products, percent_off, amount_off_cents, duration, duration_months, max_redemptions, expires_at, once_per_user, new_customers_only, active, stripe_coupon_id, stripe_promotion_code_id
FROM promo_codes
WHERE code = $1;

-- name: GetPromoCodeStripeID :one
SELECT stripe_promotion_code_id FROM promo_codes WHERE id = $1;

-- name: SetPromoCodeActive :exec
UPDATE promo_codes SET active = $2, updated_at = NOW() WHERE id = $1;

-- name: CountPromoCodeRedemptions :one
SELECT count(*)::int FROM promo_code_redemptions WHERE promo_code_id = $1 AND status = 'completed';

-- name: CountUserPromoCodeRedemptions :one
SELECT count(*)::int FROM promo_code_redemptions WHERE promo_code_id = $1 AND user_id = $2 AND status = 'completed';

-- name: ListUserPendingPromoRedemptions :many
SELECT stripe_session_id FROM promo_code_redemptions WHERE promo_code_id = $1 AND user_id = $2 AND status = 'pending';

-- name: InsertPromoCodeRedemption :exec
INSERT INTO promo_code_redemptions (promo_code_id, user_id, product, stripe_session_id)
VALUES ($1, $2, $3, $4);

-- name: CompletePromoCodeRedemption :exec
UPDATE promo_code_redemptions SET status = 'completed', completed_at = NOW()
WHERE stripe_session_id = $1 AND status <> 'completed';

-- name: AbandonPromoCodeRedemption :exec
UPDATE promo_code_redemptions SET status = 'abandoned'
WHERE stripe_session_id = $1 AND status = 'pending';

-- name: UserHasPaidBefore :one
SELECT (EXISTS (SELECT 1 FROM payment_fulfillments pf WHERE pf.user_id = sqlc.arg(user_id)::text)
    OR EXISTS (SELECT 1 FROM auto_vote_subscriptions avs WHERE avs.user_id = sqlc.arg(user_id)::text))::bool AS paid;
