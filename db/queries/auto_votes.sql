-- name: UpsertAutoVoteSubscription :exec
INSERT INTO auto_vote_subscriptions (user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, cancel_at_period_end)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    stripe_customer_id = EXCLUDED.stripe_customer_id,
    stripe_subscription_id = EXCLUDED.stripe_subscription_id,
    status = EXCLUDED.status,
    current_period_end = EXCLUDED.current_period_end,
    cancel_at_period_end = EXCLUDED.cancel_at_period_end,
    updated_at = NOW();

-- name: UpdateAutoVoteSubscriptionByStripeID :execrows
UPDATE auto_vote_subscriptions SET status = $2, current_period_end = $3, cancel_at_period_end = $4, updated_at = NOW()
WHERE stripe_subscription_id = $1;

-- name: GetAutoVoteSubscription :one
SELECT user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, cancel_at_period_end, created_at
FROM auto_vote_subscriptions WHERE user_id = $1;

-- name: ListAutoVoteTargets :many
SELECT user_id, target_type, target_id, days, start_minute, end_minute, timezone, last_voted_at, last_skip_reason, created_at
FROM auto_vote_targets WHERE user_id = $1 ORDER BY created_at;

-- name: DeleteAutoVoteTargets :exec
DELETE FROM auto_vote_targets WHERE user_id = $1;

-- name: InsertAutoVoteTarget :exec
INSERT INTO auto_vote_targets (user_id, target_type, target_id, days, start_minute, end_minute, timezone)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListDueAutoVoteTargets :many
SELECT t.user_id, t.target_type, t.target_id, t.days, t.start_minute, t.end_minute, t.timezone, t.last_voted_at, t.last_skip_reason
FROM auto_vote_targets t
INNER JOIN auto_vote_subscriptions s ON s.user_id = t.user_id
WHERE s.status IN ('active', 'trialing') AND (s.current_period_end IS NULL OR s.current_period_end > NOW());

-- name: MarkAutoVoteTargetVoted :exec
UPDATE auto_vote_targets SET last_voted_at = NOW(), last_skip_reason = NULL
WHERE user_id = $1 AND target_type = $2 AND target_id = $3;

-- name: MarkAutoVoteTargetSkipped :exec
UPDATE auto_vote_targets SET last_skip_reason = $4
WHERE user_id = $1 AND target_type = $2 AND target_id = $3;

-- name: GetBotAutoVotesOptOut :one
SELECT auto_votes_opt_out FROM bots WHERE bot_id = $1;

-- name: GetServerAutoVotesOptOut :one
SELECT auto_votes_opt_out FROM servers WHERE server_id = $1;

-- name: SetBotAutoVotesOptOut :exec
UPDATE bots SET auto_votes_opt_out = $2 WHERE bot_id = $1;

-- name: SetServerAutoVotesOptOut :exec
UPDATE servers SET auto_votes_opt_out = $2 WHERE server_id = $1;
