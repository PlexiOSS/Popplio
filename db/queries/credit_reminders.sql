-- name: ListRedeemableVoteTotals :many
SELECT target_type, target_id, (COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote))::int AS votes
FROM entity_votes
WHERE NOT void AND credit_redeem IS NULL AND NOT auto AND target_type IN ('bot', 'server')
GROUP BY target_type, target_id
HAVING COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) >= sqlc.arg(min_votes)::int;

-- name: ListUnspentCreditTotals :many
SELECT target_type, target_id, SUM(credits - redeemed_credits)::int AS credits
FROM entity_vote_redeem_logs
WHERE target_type IN ('bot', 'server')
GROUP BY target_type, target_id
HAVING SUM(credits - redeemed_credits) >= sqlc.arg(min_credits)::int;

-- name: GetCheapestShopItemPrice :one
SELECT COALESCE(MIN(cents), 0)::int AS cents FROM shop_items;

-- name: MarkCreditReminderSent :one
INSERT INTO credit_reminders (user_id, period) VALUES ($1, $2)
ON CONFLICT (user_id, period) DO NOTHING
RETURNING user_id;
