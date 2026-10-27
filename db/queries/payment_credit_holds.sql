-- name: InsertPaymentCreditHold :exec
INSERT INTO payment_credit_holds (provider, external_id, user_id, target_type, target_id, credits, discount_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: MarkPaymentCreditHoldSpent :execrows
UPDATE payment_credit_holds SET state = 'spent', resolved_at = NOW()
WHERE provider = $1 AND external_id = $2 AND state = 'held';

-- name: GetPaymentCreditHoldState :one
SELECT state FROM payment_credit_holds WHERE provider = $1 AND external_id = $2;

-- name: ListStalePaymentCreditHolds :many
SELECT id, provider, external_id, user_id, target_type, target_id, credits, created_at
FROM payment_credit_holds
WHERE state = 'held' AND created_at < sqlc.arg('before')::timestamptz
ORDER BY created_at
LIMIT 100;

-- name: MarkPaymentCreditHoldRefunded :execrows
UPDATE payment_credit_holds SET state = 'refunded', resolved_at = NOW()
WHERE id = $1 AND state = 'held';
