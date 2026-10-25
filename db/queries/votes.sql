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
SELECT name, type, vote_banned FROM servers WHERE server_id = $1;

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

-- name: GetEntityVoters :many
SELECT author,
    COUNT(*) FILTER (WHERE upvote AND NOT void) AS current_votes,
    COUNT(*) FILTER (WHERE upvote) AS total_votes,
    MAX(created_at)::timestamptz AS last_voted_at,
    COUNT(*) OVER () AS total_voters
FROM entity_votes
WHERE target_id = sqlc.arg('target_id') AND target_type = sqlc.arg('target_type')
GROUP BY author
HAVING COUNT(*) FILTER (WHERE upvote) > 0
ORDER BY current_votes DESC, total_votes DESC, last_voted_at DESC
LIMIT sqlc.arg('lim');

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

-- name: CountRedeemableEntityVotes :one
-- Same as CountEntityVotes, but additionally excludes votes that have
-- already been cashed in for credits (credit_redeem IS NOT NULL). Used only
-- by the vote-credit redemption flow -- an already-redeemed vote must not
-- count a second time toward a new redemption's total, even though it still
-- counts toward the entity's public vote total (CountEntityVotes).
SELECT COUNT(*) FILTER (WHERE upvote) AS upvotes, COUNT(*) FILTER (WHERE NOT upvote) AS downvotes
FROM entity_votes
WHERE target_id = $1 AND target_type = $2 AND void = false AND credit_redeem IS NULL;

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

-- name: LockVoteCastTarget :exec
-- A per-(user, target) advisory lock (auto-released at transaction end),
-- held from the start of create_user_entity_vote's transaction, before
-- EntityVoteCheck reads the user's existing votes/wait timer for this
-- target. Without it, two concurrent vote requests from the same user for
-- the same target can both read "not voted yet" / "wait elapsed" before
-- either commits its INSERT, both pass the PerUser/VoteTime check, and both
-- get counted -- exceeding the entity's configured vote limits. Scoped to
-- (user, target) rather than target alone so unrelated users voting for the
-- same entity at the same time are never serialized against each other.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg('user_id')::text || ':' || sqlc.arg('target_type')::text || ':' || sqlc.arg('target_id')::text));

-- name: LockVoteRedeemTarget :exec
-- A per-target advisory lock (auto-released at transaction end), NOT the
-- table-wide LockEntityVotesExclusive used by the batch vote-reset job --
-- that would serialize every redemption sitewide instead of just the ones
-- racing for the same target. Held for the duration of
-- EntityRedeemVoteCredits, before it reads the redeemable vote count, so
-- two concurrent redeem calls for the same target can no longer both read
-- "N votes still unclaimed" and both log a fresh N-vote credit grant --
-- the second call blocks until the first commits, then correctly sees
-- however many votes are actually still unclaimed (possibly zero).
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg('target_type')::text || ':' || sqlc.arg('target_id')::text));

-- name: GetRedeemableVoteItags :many
-- credit_redeem IS NULL excludes votes some earlier redemption already
-- claimed -- without it, a vote stays eligible forever and a repeat
-- redemption call would claim (and pay out credits for) the same vote more
-- than once.
SELECT itag FROM entity_votes WHERE target_id = $1 AND target_type = $2 AND void = false AND credit_redeem IS NULL ORDER BY created_at ASC LIMIT $3;

-- name: RedeemVotesByItags :exec
-- Deliberately does NOT set void = true. A vote being spent on credits is
-- not the same event as a vote being reset/removed (VoidAllUnvoidedEntityVotes,
-- VoidEntityVotesForTarget, VoidAllEntityVotesForType all mean the latter) --
-- voiding a redeemed vote used to also drop it from the entity's public vote
-- count (every count query filters on void = false), silently deflating a
-- bot/server/team's vote total every time its owner cashed in credits.
-- credit_redeem (set here) is what actually marks a vote as spent; void_reason/
-- voided_at are kept purely as display metadata for the vote history UI, not
-- as a gate on anything. credit_redeem IS NULL guards the same re-claim
-- risk RedeemAllVotesForTarget's own comment explains -- belt-and-suspenders
-- alongside the caller's per-target advisory lock (LockVoteRedeemTarget),
-- in case a future caller ever reaches this query without going through it.
UPDATE entity_votes SET credit_redeem = $1, void_reason = 'Vote credits redeemed', voided_at = NOW() WHERE itag = ANY(sqlc.arg('itags')::uuid[]) AND credit_redeem IS NULL;

-- name: RedeemAllVotesForTarget :exec
-- See RedeemVotesByItags -- same fix, same reasoning. credit_redeem IS NULL
-- in the WHERE clause is required now that redeeming no longer voids a vote:
-- without it, a vote already spent by an earlier redemption (void = false,
-- credit_redeem set) would still match "void = false" and get claimed again.
UPDATE entity_votes SET credit_redeem = $1, void_reason = 'Vote credits redeemed', voided_at = NOW() WHERE target_id = $2 AND target_type = $3 AND void = false AND credit_redeem IS NULL;

-- name: GetVoteRedeemLogs :many
SELECT target_id, target_type, created_at, redeemed_at, id, redeemed_credits, credits
FROM entity_vote_redeem_logs
WHERE target_id = $1 AND target_type = $2
ORDER BY created_at DESC;

-- name: GetTopVoters :many
WITH counted AS (
    SELECT ev.author,
        COUNT(*) AS vote_count,
        COUNT(*) FILTER (WHERE ev.target_type = 'bot') AS bot_votes,
        COUNT(*) FILTER (WHERE ev.target_type = 'server') AS server_votes,
        COUNT(*) FILTER (WHERE ev.target_type NOT IN ('bot', 'server')) AS other_votes,
        MIN(ev.created_at) AS first_vote
    FROM entity_votes ev
    WHERE ev.upvote = true
    AND ev.void_reason IS DISTINCT FROM 'Votes (single entity) reset'
    AND (sqlc.narg('since')::timestamptz IS NULL OR ev.created_at >= sqlc.narg('since')::timestamptz)
    AND NOT EXISTS (SELECT 1 FROM users u WHERE u.user_id = ev.author AND (u.vote_banned OR u.banned))
    GROUP BY ev.author
)
SELECT author, vote_count, bot_votes, server_votes, other_votes,
    ROW_NUMBER() OVER (ORDER BY vote_count DESC, first_vote ASC, author ASC)::bigint AS rank
FROM counted
ORDER BY rank
LIMIT sqlc.arg('lim');

-- name: GetVoterRank :one
WITH counted AS (
    SELECT ev.author,
        COUNT(*) AS vote_count,
        MIN(ev.created_at) AS first_vote
    FROM entity_votes ev
    WHERE ev.upvote = true
    AND ev.void_reason IS DISTINCT FROM 'Votes (single entity) reset'
    AND (sqlc.narg('since')::timestamptz IS NULL OR ev.created_at >= sqlc.narg('since')::timestamptz)
    AND NOT EXISTS (SELECT 1 FROM users u WHERE u.user_id = ev.author AND (u.vote_banned OR u.banned))
    GROUP BY ev.author
), ranked AS (
    SELECT author, vote_count,
        ROW_NUMBER() OVER (ORDER BY vote_count DESC, first_vote ASC, author ASC)::bigint AS rank,
        COUNT(*) OVER ()::bigint AS total_voters
    FROM counted
)
SELECT rank, vote_count, total_voters
FROM ranked
WHERE author = sqlc.arg('author');

-- name: GetVoterFavorites :many
SELECT DISTINCT ON (ev.author) ev.author, ev.target_type, ev.target_id, COUNT(*) AS vote_count,
    COALESCE(s.name, '')::text AS server_name, COALESCE(s.avatar, '')::text AS server_avatar, COALESCE(v.code::text, '')::text AS vanity
FROM entity_votes ev
LEFT JOIN bots b ON ev.target_type = 'bot' AND b.bot_id = ev.target_id
LEFT JOIN servers s ON ev.target_type = 'server' AND s.server_id = ev.target_id
LEFT JOIN vanity v ON v.itag = COALESCE(b.vanity_ref, s.vanity_ref)
WHERE ev.author = ANY(sqlc.arg('authors')::text[])
AND ev.upvote = true
AND ev.void_reason IS DISTINCT FROM 'Votes (single entity) reset'
AND (sqlc.narg('since')::timestamptz IS NULL OR ev.created_at >= sqlc.narg('since')::timestamptz)
AND (
    (b.bot_id IS NOT NULL AND b.type IN ('approved', 'certified') AND NOT b.nsfw)
    OR (s.server_id IS NOT NULL AND s.type IN ('approved', 'certified') AND s.state = 'public' AND NOT s.nsfw)
)
GROUP BY ev.author, ev.target_type, ev.target_id, s.name, s.avatar, v.code
ORDER BY ev.author, COUNT(*) DESC, MAX(ev.created_at) DESC;

-- name: GetLatestAutomatedVoteReset :one
SELECT created_at FROM automated_vote_resets ORDER BY created_at DESC LIMIT 1;

-- name: GetUserTopVoted :many
SELECT ev.target_type, ev.target_id, COUNT(*) AS vote_count,
    COALESCE(s.name, '')::text AS server_name, COALESCE(s.avatar, '')::text AS server_avatar, COALESCE(v.code::text, '')::text AS vanity
FROM entity_votes ev
LEFT JOIN bots b ON ev.target_type = 'bot' AND b.bot_id = ev.target_id
LEFT JOIN servers s ON ev.target_type = 'server' AND s.server_id = ev.target_id
LEFT JOIN vanity v ON v.itag = COALESCE(b.vanity_ref, s.vanity_ref)
WHERE ev.author = sqlc.arg('author')
AND ev.upvote = true
AND ev.void_reason IS DISTINCT FROM 'Votes (single entity) reset'
AND ((b.bot_id IS NOT NULL AND b.type IN ('approved', 'certified') AND NOT b.nsfw)
    OR (s.server_id IS NOT NULL AND s.type IN ('approved', 'certified') AND s.state = 'public' AND NOT s.nsfw))
GROUP BY ev.target_type, ev.target_id, s.name, s.avatar, v.code
ORDER BY COUNT(*) DESC, MAX(ev.created_at) DESC
LIMIT sqlc.arg('lim');
