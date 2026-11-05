-- The target of a premium purchase is always either a bot or a server
-- (validated up front by entityTarget in routes/payments/assets/give_perks.go),
-- never an unvalidated table name -- but sqlc needs a static table per query,
-- so this is one explicit pair per entity type rather than one
-- dynamically-templated query.

-- name: CountBotByID :one
SELECT COUNT(*) FROM bots WHERE bot_id = $1;

-- name: CountServerByID :one
SELECT COUNT(*) FROM servers WHERE server_id = $1;

-- name: GetBotTypePremium :one
SELECT type, premium FROM bots WHERE bot_id = $1;

-- name: GetServerTypePremium :one
SELECT type, premium FROM servers WHERE server_id = $1;


-- name: InsertPaymentFulfillment :execrows
INSERT INTO payment_fulfillments (provider, external_id, user_id, product_id, for_type, for_id, premium_hours)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (provider, external_id) DO NOTHING;

-- name: GetFirstPaymentFulfillmentAt :one
SELECT MIN(fulfilled_at)::timestamptz FROM payment_fulfillments WHERE provider = $1;

-- name: PaymentFulfillmentExists :one
SELECT EXISTS(SELECT 1 FROM payment_fulfillments WHERE provider = $1 AND external_id = $2);

-- name: GetPaymentFulfillment :one
SELECT provider, external_id, user_id, product_id, for_type, for_id, fulfilled_at, revoked_at, revoke_reason, premium_hours
FROM payment_fulfillments WHERE provider = $1 AND external_id = $2;

-- name: MarkPaymentFulfillmentRevoked :execrows
UPDATE payment_fulfillments SET revoked_at = NOW(), revoke_reason = $3
WHERE provider = $1 AND external_id = $2 AND revoked_at IS NULL;

-- name: RevokeBotPremiumFrom :one
UPDATE bots SET
    premium_period_length = CASE WHEN sqlc.narg(hours)::int IS NULL THEN interval '0' ELSE GREATEST(premium_period_length - make_interval(hours => sqlc.narg(hours)::int), interval '0') END,
    premium = sqlc.narg(hours)::int IS NOT NULL AND start_premium_period + premium_period_length - make_interval(hours => sqlc.narg(hours)::int) > NOW()
WHERE bot_id = sqlc.arg(bot_id) AND premium = true AND start_premium_period = sqlc.arg(granted_at)::timestamptz
RETURNING premium;

-- name: RevokeServerPremiumFrom :one
UPDATE servers SET
    premium_period_length = CASE WHEN sqlc.narg(hours)::int IS NULL THEN interval '0' ELSE GREATEST(premium_period_length - make_interval(hours => sqlc.narg(hours)::int), interval '0') END,
    premium = sqlc.narg(hours)::int IS NOT NULL AND start_premium_period + premium_period_length - make_interval(hours => sqlc.narg(hours)::int) > NOW()
WHERE server_id = sqlc.arg(server_id) AND premium = true AND start_premium_period = sqlc.arg(granted_at)::timestamptz
RETURNING premium;

-- name: ListRecentPaymentFulfillments :many
SELECT f.provider, f.external_id, f.user_id, f.product_id, f.for_type, f.for_id, f.fulfilled_at, f.revoked_at, f.revoke_reason, f.premium_hours,
    h.credits AS credits_used, h.discount_cents AS credit_discount_cents, h.state AS credit_state,
    p.code AS promo_code
FROM payment_fulfillments f
LEFT JOIN payment_credit_holds h ON h.provider = f.provider AND h.external_id = f.external_id
LEFT JOIN promo_code_redemptions r ON r.stripe_session_id = f.external_id AND r.status = 'completed'
LEFT JOIN promo_codes p ON p.id = r.promo_code_id
WHERE (sqlc.narg(user_id)::text IS NULL OR f.user_id = sqlc.narg(user_id)::text OR f.for_id = sqlc.narg(user_id)::text)
  AND (NOT sqlc.arg(revoked_only)::bool OR f.revoked_at IS NOT NULL)
ORDER BY f.fulfilled_at DESC
LIMIT sqlc.arg(lim)::int;
