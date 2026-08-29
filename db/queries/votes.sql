-- Entity-target dispatch queries (popplio/votes.GetEntityInfo, EntityVoteInfo,
-- EntityPostVote, RecomputeApproximateVotes each pick one of a fixed,
-- never-user-controlled set of tables based on target_type) -- one explicit
-- query per table, same reasoning as db/queries/payments.sql.

-- name: GetBotVoteStatus :one
SELECT type, vote_banned FROM bots WHERE bot_id = $1;

-- name: GetPackVoteBanned :one
SELECT vote_banned FROM packs WHERE url = $1;

-- name: GetTeamVoteStatus :one
SELECT name, vote_banned FROM teams WHERE id = $1;

-- name: GetServerVoteStatus :one
SELECT name, vote_banned FROM servers WHERE server_id = $1;

-- name: GetBotVoteInfo :one
SELECT premium, vote_blitz_until FROM bots WHERE bot_id = $1;

-- name: GetServerVoteInfo :one
SELECT premium, vote_blitz_until FROM servers WHERE server_id = $1;

-- name: UpdateBotApproximateVotes :exec
UPDATE bots SET approximate_votes = $1 WHERE bot_id = $2;

-- name: UpdateServerApproximateVotes :exec
UPDATE servers SET approximate_votes = $1 WHERE server_id = $2;

-- name: UpdateTeamApproximateVotes :exec
UPDATE teams SET approximate_votes = $1 WHERE id = $2;

-- name: ZeroBotApproximateVotes :exec
UPDATE bots SET approximate_votes = 0;

-- name: ZeroServerApproximateVotes :exec
UPDATE servers SET approximate_votes = 0;

-- name: ZeroTeamApproximateVotes :exec
UPDATE teams SET approximate_votes = 0;

-- name: RecomputeBotApproximateVotes :exec
UPDATE bots t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = $1 AND void = false GROUP BY target_id) v
WHERE t.bot_id = v.target_id;

-- name: RecomputeServerApproximateVotes :exec
UPDATE servers t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = $1 AND void = false GROUP BY target_id) v
WHERE t.server_id = v.target_id;

-- name: RecomputeTeamApproximateVotes :exec
UPDATE teams t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = $1 AND void = false GROUP BY target_id) v
WHERE t.id::text = v.target_id;

-- name: GetUserEntityVotesPage :many
SELECT itag, target_id, target_type, author, upvote, void, void_reason, voided_at, created_at, vote_num, credit_redeem, immutable
FROM entity_votes
WHERE target_id = $1 AND target_type = $2 AND author = $3
LIMIT $4 OFFSET $5;

-- name: CountUserEntityVotes :one
SELECT COUNT(*) FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND author = $3;

-- name: GetUserVoteBanned :one
SELECT vote_banned FROM users WHERE user_id = $1;

-- name: DeleteUserEntityVotes :exec
DELETE FROM entity_votes WHERE author = $1 AND target_id = $2 AND target_type = $3;

-- name: GetVoteAuthors :many
SELECT author FROM entity_votes WHERE target_id = $1 AND target_type = $2 GROUP BY author;

-- name: GetRecentAutomatedVoteResetForUpdate :one
SELECT id FROM automated_vote_resets WHERE created_at > NOW() - INTERVAL '1 month' FOR UPDATE;

-- name: LockEntityVotesExclusive :exec
LOCK TABLE entity_votes IN EXCLUSIVE MODE;

-- name: VoidAllUnvoidedEntityVotes :exec
UPDATE entity_votes SET void = TRUE, void_reason = 'Automated votes reset', voided_at = NOW() WHERE void = false AND immutable = false;

-- name: InsertAutomatedVoteReset :exec
INSERT INTO automated_vote_resets (created_at) VALUES (NOW());

-- name: VoidEntityVotesForTarget :exec
UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (single entity) reset', voided_at = NOW()
WHERE target_type = $1 AND target_id = $2 AND void = FALSE;

-- name: VoidAllEntityVotesForType :exec
UPDATE entity_votes SET void = TRUE, void_reason = 'Votes (all entities) reset', voided_at = NOW()
WHERE target_type = $1 AND immutable = false;

-- name: GetServerVoteLeaderboard :many
SELECT author,
(SUM(CASE WHEN upvote THEN 1 ELSE 0 END) - SUM(CASE WHEN NOT upvote THEN 1 ELSE 0 END))::bigint AS score
FROM entity_votes
WHERE target_id = $1
AND target_type = 'server'
AND void = false
GROUP BY author
ORDER BY score DESC;

-- name: GetVoteAuthorsPage :many
SELECT author FROM entity_votes WHERE target_id = $1 AND target_type = $2 GROUP BY author LIMIT $3 OFFSET $4;

-- name: GetEntityVotes :many
SELECT itag, target_id, target_type, author, upvote, void, void_reason, voided_at, created_at, vote_num, credit_redeem, immutable
FROM entity_votes
WHERE author = $1 AND target_id = $2 AND target_type = $3 AND void = false
ORDER BY created_at DESC;

-- name: CountEntityVotes :one
SELECT COUNT(*) FILTER (WHERE upvote) AS upvotes, COUNT(*) FILTER (WHERE NOT upvote) AS downvotes
FROM entity_votes
WHERE target_id = $1 AND target_type = $2 AND void = false;

-- name: InsertEntityVote :exec
INSERT INTO entity_votes (author, target_id, target_type, upvote, vote_num) VALUES ($1, $2, $3, $4, $5);

-- name: GetVoteCreditTiers :many
SELECT id, target_type, position, votes, cents, created_at FROM vote_credit_tiers WHERE target_type = $1 ORDER BY position ASC;

-- name: GetVoteCreditTiersFiltered :many
SELECT id, target_type, position, votes, cents, created_at
FROM vote_credit_tiers
WHERE sqlc.narg('target_type')::text IS NULL OR target_type = sqlc.narg('target_type')
ORDER BY position ASC;

-- name: InsertVoteRedeemLog :one
INSERT INTO entity_vote_redeem_logs (target_id, target_type, credits) VALUES ($1, $2, $3) RETURNING id;

-- name: GetRedeemableVoteItags :many
SELECT itag FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false ORDER BY created_at ASC LIMIT $3;

-- name: RedeemVotesByItags :exec
UPDATE entity_votes SET credit_redeem = $1, void = true, void_reason = 'Vote credits redeemed' WHERE itag = ANY(sqlc.arg('itags')::uuid[]);

-- name: RedeemAllVotesForTarget :exec
UPDATE entity_votes SET credit_redeem = $1, void = true, void_reason = 'Vote credits redeemed' WHERE target_id = $2 AND target_type = $3 AND void = false;

-- name: GetVoteRedeemLogs :many
SELECT target_id, target_type, created_at, redeemed_at, id, redeemed_credits, credits
FROM entity_vote_redeem_logs
WHERE target_id = $1 AND target_type = $2
ORDER BY created_at DESC;
