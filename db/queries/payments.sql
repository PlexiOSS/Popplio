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
INSERT INTO payment_fulfillments (provider, external_id, user_id, product_id, for_type, for_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, external_id) DO NOTHING;

-- name: GetFirstPaymentFulfillmentAt :one
SELECT MIN(fulfilled_at)::timestamptz FROM payment_fulfillments WHERE provider = $1;

-- name: PaymentFulfillmentExists :one
SELECT EXISTS(SELECT 1 FROM payment_fulfillments WHERE provider = $1 AND external_id = $2);
