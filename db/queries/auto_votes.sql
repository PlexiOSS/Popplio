-- name: UpsertAutoVoteSubscription :exec
INSERT INTO auto_vote_subscriptions (user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, cancel_at_period_end, extra_slots, billing_interval)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id) DO UPDATE SET
    stripe_customer_id = EXCLUDED.stripe_customer_id,
    stripe_subscription_id = EXCLUDED.stripe_subscription_id,
    status = EXCLUDED.status,
    current_period_end = EXCLUDED.current_period_end,
    cancel_at_period_end = EXCLUDED.cancel_at_period_end,
    extra_slots = EXCLUDED.extra_slots,
    billing_interval = EXCLUDED.billing_interval,
    updated_at = NOW();

-- name: UpdateAutoVoteSubscriptionByStripeID :execrows
UPDATE auto_vote_subscriptions SET status = $2, current_period_end = $3, cancel_at_period_end = $4, extra_slots = $5, billing_interval = $6, updated_at = NOW()
WHERE stripe_subscription_id = $1;

-- name: SetAutoVoteExtraSlots :exec
UPDATE auto_vote_subscriptions SET extra_slots = $2, updated_at = NOW() WHERE user_id = $1;

-- name: GetAutoVoteSubscription :one
SELECT user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, cancel_at_period_end, created_at, extra_slots, billing_interval
FROM auto_vote_subscriptions WHERE user_id = $1;

-- name: GetAutoVoteExtraSlots :one
SELECT COALESCE((SELECT extra_slots FROM auto_vote_subscriptions WHERE user_id = $1), 0)::int;

-- name: ListAutoVoteTargets :many
SELECT user_id, target_type, target_id, days, start_minute, end_minute, timezone, last_voted_at, last_skip_reason, created_at, paused
FROM auto_vote_targets WHERE user_id = $1 ORDER BY created_at;
-- name: DeleteAutoVoteTargetsExcept :exec
DELETE FROM auto_vote_targets
WHERE user_id = sqlc.arg(user_id) AND NOT (target_type || ':' || target_id = ANY(sqlc.arg(keep)::text[]));

-- name: UpsertAutoVoteTarget :exec
INSERT INTO auto_vote_targets (user_id, target_type, target_id, days, start_minute, end_minute, timezone, paused)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id, target_type, target_id) DO UPDATE SET
    days = EXCLUDED.days,
    start_minute = EXCLUDED.start_minute,
    end_minute = EXCLUDED.end_minute,
    timezone = EXCLUDED.timezone,
    paused = EXCLUDED.paused;
-- name: ListDueAutoVoteTargets :many
SELECT t.user_id, t.target_type, t.target_id, t.days, t.start_minute, t.end_minute, t.timezone, t.last_voted_at, t.last_skip_reason,
    COALESCE(st.notify, 'off')::text AS notify
FROM (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at, target_type, target_id) AS slot
    FROM auto_vote_targets
) t
INNER JOIN auto_vote_subscriptions s ON s.user_id = t.user_id
LEFT JOIN auto_vote_settings st ON st.user_id = t.user_id
LEFT JOIN booster_perks b ON b.user_id = t.user_id
WHERE NOT t.paused AND t.slot <= 3 + s.extra_slots + CASE WHEN COALESCE(b.boosting, false) THEN 1 ELSE 0 END AND s.status IN ('active', 'trialing') AND (s.current_period_end IS NULL OR s.current_period_end > NOW());

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

-- name: ListAllAutoVoteSubscriptions :many
SELECT user_id, stripe_customer_id, stripe_subscription_id, status, current_period_end, cancel_at_period_end, created_at, updated_at, extra_slots, billing_interval
FROM auto_vote_subscriptions ORDER BY created_at DESC;

-- name: ListAllAutoVoteTargets :many
SELECT user_id, target_type, target_id, days, start_minute, end_minute, timezone, last_voted_at, last_skip_reason, created_at, paused
FROM auto_vote_targets ORDER BY user_id, created_at;
-- name: CountAutoVotesSince :many
SELECT author, count(*)::int AS votes FROM entity_votes
WHERE auto AND NOT void AND created_at > sqlc.arg(since)::timestamp
GROUP BY author;

-- name: GetAutoVoteNotify :one
SELECT notify FROM auto_vote_settings WHERE user_id = $1;

-- name: SetAutoVoteNotify :exec
INSERT INTO auto_vote_settings (user_id, notify, last_summary_at)
VALUES (sqlc.arg(user_id), sqlc.arg(notify), CASE WHEN sqlc.arg(notify)::text = 'daily' THEN NOW() END)
ON CONFLICT (user_id) DO UPDATE SET
    notify = EXCLUDED.notify,
    last_summary_at = CASE
        WHEN EXCLUDED.notify = 'daily' AND auto_vote_settings.notify <> 'daily' THEN NOW()
        ELSE auto_vote_settings.last_summary_at
    END,
    updated_at = NOW();

-- name: ListDueAutoVoteSummaries :many
SELECT st.user_id, st.last_summary_at
FROM auto_vote_settings st
INNER JOIN auto_vote_subscriptions s ON s.user_id = st.user_id
WHERE st.notify = 'daily' AND s.status IN ('active', 'trialing')
    AND (st.last_summary_at IS NULL OR st.last_summary_at < NOW() - INTERVAL '24 hours');

-- name: CountUserAutoVotesSince :many
SELECT target_type, target_id, count(*)::int AS votes
FROM entity_votes
WHERE author = sqlc.arg(author) AND auto AND NOT void AND created_at > sqlc.arg(since)::timestamp
GROUP BY target_type, target_id
ORDER BY votes DESC;

-- name: MarkAutoVoteSummarySent :exec
UPDATE auto_vote_settings SET last_summary_at = NOW() WHERE user_id = $1;

-- name: SetAutoVoteCardFingerprint :execrows
UPDATE auto_vote_subscriptions SET card_fingerprint = $2, updated_at = NOW()
WHERE stripe_subscription_id = $1 AND card_fingerprint IS DISTINCT FROM $2;

-- name: ListAutoVoteAccountsOnCard :many
SELECT user_id FROM auto_vote_subscriptions
WHERE card_fingerprint = $1 AND status IN ('active', 'trialing') AND (current_period_end IS NULL OR current_period_end > NOW())
ORDER BY created_at, user_id;

-- name: ListAutoVoteUsersOverCardLimit :many
SELECT user_id FROM (
    SELECT user_id, ROW_NUMBER() OVER (PARTITION BY card_fingerprint ORDER BY created_at, user_id) AS position
    FROM auto_vote_subscriptions
    WHERE card_fingerprint IS NOT NULL AND status IN ('active', 'trialing') AND (current_period_end IS NULL OR current_period_end > NOW())
) ranked
WHERE position > sqlc.arg(max_accounts)::int;

-- name: ListAutoVoteShareSpikes :many
SELECT target_type, target_id, auto_day, total_day, auto_before, total_before, auto_voters FROM (
    SELECT target_type, target_id,
        COUNT(*) FILTER (WHERE auto AND created_at > NOW() - INTERVAL '1 day')::int AS auto_day,
        COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day')::int AS total_day,
        COUNT(*) FILTER (WHERE auto AND created_at <= NOW() - INTERVAL '1 day')::int AS auto_before,
        COUNT(*) FILTER (WHERE created_at <= NOW() - INTERVAL '1 day')::int AS total_before,
        COUNT(DISTINCT author) FILTER (WHERE auto AND created_at > NOW() - INTERVAL '1 day')::int AS auto_voters
    FROM entity_votes
    WHERE upvote AND NOT void AND created_at > NOW() - INTERVAL '8 days' AND target_type IN ('bot', 'server')
    GROUP BY target_type, target_id
) v
WHERE auto_day >= sqlc.arg(min_auto)::int;
