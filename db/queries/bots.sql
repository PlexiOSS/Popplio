-- IndexBot is the reduced projection used for listings, search results, and
-- embedding a bot inside other entities (packs, etc.) -- shared across
-- several packages, hence its own file even though "bots" the full package
-- hasn't been converted yet.

-- name: GetIndexBotByID :one
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE bot_id = $1;

-- name: SearchBotsPublic :many
-- Backs POST /list/search's "bot" target type. iucd is the Discord user
-- cache (internal_user_cache__discord) -- left-joined unconditionally so
-- bots with no cached user row still show up when there's no text query,
-- same as the raw-SQL version this replaced only inner-joining when needed.
SELECT bots.bot_id, bots.short, bots.type, bots.vanity_ref, bots.approximate_votes, bots.shards, bots.library,
    bots.invite_clicks, bots.clicks, bots.servers, bots.nsfw, bots.tags, bots.premium, bots.created_at,
    bots.self_status, bots.last_stats_post, bots.supporter_badge, bots.boosted_until, bots.featured_until,
    bots.spotlighted_until, bots.vote_blitz_until
FROM bots
LEFT JOIN internal_user_cache__discord iucd ON bots.bot_id = iucd.id
WHERE (bots.type = 'approved' OR bots.type = 'certified')
AND (sqlc.arg('servers_from')::int = 0 OR bots.servers >= sqlc.arg('servers_from')::int)
AND (sqlc.arg('servers_to')::int = 0 OR bots.servers <= sqlc.arg('servers_to')::int)
AND (sqlc.arg('votes_from')::int = 0 OR bots.approximate_votes >= sqlc.arg('votes_from')::int)
AND (sqlc.arg('votes_to')::int = 0 OR bots.approximate_votes <= sqlc.arg('votes_to')::int)
AND (sqlc.arg('shards_from')::int = 0 OR bots.shards >= sqlc.arg('shards_from')::int)
AND (sqlc.arg('shards_to')::int = 0 OR bots.shards <= sqlc.arg('shards_to')::int)
AND (
    cardinality(sqlc.arg('tags')::text[]) = 0
    OR (sqlc.arg('tag_mode')::text = '@>' AND bots.tags @> sqlc.arg('tags')::text[])
    OR (sqlc.arg('tag_mode')::text = '&&' AND bots.tags && sqlc.arg('tags')::text[])
)
AND (
    sqlc.arg('query')::text = ''
    OR bots.short @@ sqlc.arg('query')::text
    OR bots.bot_id = sqlc.arg('query')::text
    OR bots.client_id = sqlc.arg('query')::text
    OR iucd.username @@ sqlc.arg('query')::text
    OR iucd.username ILIKE sqlc.arg('pattern')::text
)
ORDER BY bots.approximate_votes DESC, bots.type DESC
LIMIT 12;

-- name: GetBotsDueForModerationScan :many
SELECT bot_id, short, long, nsfw
FROM bots
WHERE type IN ('approved', 'certified', 'pending')
AND (moderation_checked_at IS NULL OR moderation_checked_at < NOW() - INTERVAL '7 days')
ORDER BY moderation_checked_at ASC NULLS FIRST
LIMIT $1;

-- name: RecordBotModerationScan :exec
UPDATE bots SET moderation_flagged = $2, moderation_categories = $3, moderation_checked_at = NOW() WHERE bot_id = $1;

-- name: GetBotInvite :one
SELECT invite FROM bots WHERE bot_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: FixNoneClaimedBots :exec
UPDATE bots SET claimed_by = NULL, type = 'pending' WHERE LOWER(claimed_by) = 'none';

-- name: GetStaleClaimedBotsForUpdate :many
SELECT bot_id, claimed_by, last_claimed FROM bots WHERE claimed_by IS NOT NULL AND NOW() - last_claimed > INTERVAL '1 hour' FOR UPDATE;

-- name: GetAllBotIDs :many
SELECT bot_id FROM bots;

-- name: GetExpiredPremiumBots :many
SELECT bot_id, start_premium_period, premium_period_length, type FROM bots
WHERE (
	premium = true
	AND (
		(type != 'approved' AND type != 'certified')
		OR (start_premium_period + premium_period_length) < NOW()
	)
);

-- name: GetBotsDueForJapiUpdate :many
-- Includes pending/under_review, not just approved/certified -- a bot
-- resolved via Discord's RPC endpoint at submission time could sit with a
-- stale/zero guild count for as long as it's awaiting review otherwise,
-- since this is the only thing that ever refreshes it.
SELECT bot_id FROM bots
WHERE type IN ('approved', 'certified', 'pending', 'under_review')
AND (last_stats_post IS NULL OR NOW() - last_stats_post > INTERVAL '3 days')
AND (last_japi_update IS NULL OR NOW() - last_japi_update > INTERVAL '3 days')
ORDER BY RANDOM() LIMIT 10;

-- name: UpdateBotJapiServers :exec
UPDATE bots SET last_japi_update = NOW(), servers = $1 WHERE bot_id = $2;

-- name: TouchBotJapiUpdate :exec
UPDATE bots SET last_japi_update = NOW() WHERE bot_id = $1;

-- name: GetListedBotIDs :many
SELECT bot_id FROM bots WHERE type = 'approved' OR type = 'certified';

-- name: RecordBotUptimeCheck :exec
UPDATE bots SET
	total_uptime = total_uptime + 1,
	uptime = uptime + CASE WHEN sqlc.arg('online')::boolean THEN 1 ELSE 0 END,
	uptime_last_checked = NOW()
WHERE bot_id = sqlc.arg('bot_id')::text;

-- name: ResubmitBot :exec
UPDATE bots SET type = 'pending', claimed_by = NULL WHERE bot_id = $1;

-- name: GetBotCertStats :one
SELECT type, servers, cardinality(unique_clicks) AS unique_clicks, approximate_votes, created_at
FROM bots
WHERE bot_id = $1;

-- name: CertifyBot :exec
UPDATE bots SET type = 'certified' WHERE bot_id = $1;

-- name: GetBotCaptchaOptOut :one
SELECT captcha_opt_out FROM bots WHERE bot_id = $1;

-- name: GetBotSEOFields :one
SELECT short, owner, team_owner, created_at, updated_at
FROM bots
WHERE bot_id = $1 AND (type = 'approved' OR type = 'certified');

-- name: GetIndexBotsByTeamOwner :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE team_owner = $1;

-- name: GetBotTypeAndClaimedBy :one
SELECT type, claimed_by FROM bots WHERE bot_id = $1;

-- Panel bot review queue and search (arcadia/panel ops_queue.go, ops_search.go).

-- name: BotQueuePending :many
SELECT bot_id, client_id, last_claimed, claimed_by, type, approval_note, short,
       invite, approximate_votes, shards, library, invite_clicks, clicks, servers,
       moderation_flagged, moderation_categories
FROM bots WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at;

-- name: SearchBotsQueue :many
SELECT bots.bot_id, bots.client_id, bots.type, bots.approximate_votes, bots.shards, bots.library, bots.invite_clicks, bots.clicks,
       bots.servers, bots.last_claimed, bots.claimed_by, bots.approval_note, bots.short, bots.invite,
       bots.moderation_flagged, bots.moderation_categories
FROM bots
INNER JOIN internal_user_cache__discord discord_users ON bots.bot_id = discord_users.id
WHERE bots.bot_id = sqlc.arg('query')::text OR bots.client_id = sqlc.arg('query')::text OR discord_users.username ILIKE sqlc.arg('pattern')::text
ORDER BY bots.created_at;

-- name: UpdateBotClaim :exec
UPDATE bots SET last_claimed = NOW(), claimed_by = $1 WHERE bot_id = $2;

-- name: UncertifyBot :exec
UPDATE bots SET type = 'approved' WHERE bot_id = $1;

-- name: ClearBotFeaturedUntil :exec
UPDATE bots SET featured_until = NULL WHERE bot_id = $1;

-- name: SetBotSpotlightedUntil :exec
UPDATE bots SET spotlighted_until = GREATEST(COALESCE(spotlighted_until, NOW()), NOW()) + make_interval(hours => sqlc.arg('hours')::int) WHERE bot_id = sqlc.arg('bot_id')::text;

-- name: ClearBotSpotlightedUntil :exec
UPDATE bots SET spotlighted_until = NULL WHERE bot_id = $1;

-- name: RemoveBotPremium :exec
UPDATE bots SET premium = false WHERE bot_id = $1;

-- name: UpdateBotOwner :exec
UPDATE bots SET owner = $2 WHERE bot_id = $1;

-- name: SetBotTeamOwnerDirect :exec
UPDATE bots SET team_owner = $2 WHERE bot_id = $1;

-- name: GetBotClientID :one
SELECT client_id FROM bots WHERE bot_id = $1;

-- name: GetBotReviewStatusWithOwner :one
SELECT type, claimed_by, owner, last_claimed FROM bots WHERE bot_id = $1;

-- name: DenyBot :exec
UPDATE bots SET type = 'denied', claimed_by = NULL WHERE bot_id = $1;

-- name: GetBotReviewStatus :one
SELECT type, claimed_by, last_claimed FROM bots WHERE bot_id = $1;

-- name: ApproveBot :exec
UPDATE bots SET type = 'approved', claimed_by = NULL WHERE bot_id = $1;

-- name: SetBotVoteBanned :exec
UPDATE bots SET vote_banned = $2 WHERE bot_id = $1;

-- name: GetBotTeamAndOwner :one
SELECT team_owner, owner FROM bots WHERE bot_id = $1;

-- name: GetBotType :one
SELECT type FROM bots WHERE bot_id = $1;

-- name: GetIndexBotsByOwner :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE owner = $1;

-- name: GetIndexBotsByTeamMembership :many
-- A bot's owner and team_owner are mutually exclusive (TransferToTeam
-- clears owner when setting team_owner), so this and GetIndexBotsByOwner
-- never return the same row -- callers union both, same pattern
-- GetIndexServersByTeamMembership already established for servers.
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE team_owner IN (SELECT team_id FROM team_members WHERE user_id = $1);

-- name: GetBotSEO :one
SELECT short FROM bots WHERE bot_id = $1;

-- name: GetBotByID :one
SELECT itag, bot_id, client_id, extra_links, tags, prefix, owner, short, library, nsfw, premium, last_stats_post, last_japi_update, servers, shards, shard_list, users, approximate_votes, clicks, invite_clicks, invite, type, vanity_ref, vote_banned, start_premium_period, premium_period_length, cert_reason, uptime, total_uptime, uptime_last_checked, approval_note, created_at, claimed_by, updated_at, last_claimed, team_owner, captcha_opt_out, self_status, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until, moderation_flagged, moderation_categories
FROM bots
WHERE bot_id = $1;

-- name: GetBotUniqueClicksCount :one
SELECT cardinality(unique_clicks) FROM bots WHERE bot_id = $1;

-- name: UpdateBotClicks :exec
UPDATE bots SET clicks = clicks + 1 WHERE bot_id = $1;

-- name: CheckBotHasUniqueClick :one
SELECT sqlc.arg('hashed_ip')::text = ANY(unique_clicks) FROM bots WHERE bot_id = sqlc.arg('bot_id')::text;

-- name: AppendBotUniqueClick :exec
UPDATE bots SET unique_clicks = array_append(unique_clicks, $1) WHERE bot_id = $2;

-- name: UpdateBotInviteClicks :exec
UPDATE bots SET invite_clicks = invite_clicks + 1 WHERE bot_id = $1;

-- name: GetBotLongDescription :one
SELECT long FROM bots WHERE bot_id = $1;

-- name: GetRandomIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE (type = 'approved' OR type = 'certified')
ORDER BY RANDOM()
LIMIT 6;

-- name: GetSimilarBots :many
-- Tag-overlap similarity: other approved/certified bots sharing at least
-- one tag with bot_id, ranked by how many tags they share (most first),
-- votes as the tiebreak. No ML/embeddings -- tags are the only structured
-- signal bots already carry, and it's cheap, deterministic, and explains
-- itself ("shares N tags with this bot").
WITH source_tags AS (
    SELECT tags FROM bots WHERE bot_id = sqlc.arg('bot_id')
)
SELECT b.bot_id, b.short, b.type, b.vanity_ref, b.approximate_votes, b.shards, b.library, b.invite_clicks, b.clicks, b.servers, b.nsfw, b.tags, b.premium, b.created_at, b.self_status, b.last_stats_post, b.supporter_badge, b.boosted_until, b.featured_until, b.spotlighted_until, b.vote_blitz_until
FROM bots b, source_tags st
WHERE (b.type = 'approved' OR b.type = 'certified')
AND b.bot_id != sqlc.arg('bot_id')
AND b.tags && st.tags
ORDER BY cardinality(ARRAY(SELECT unnest(b.tags) INTERSECT SELECT unnest(st.tags))) DESC, b.approximate_votes DESC
LIMIT 6;

-- name: GetIndexBotsPaged :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE (type = 'approved' OR type = 'certified')
ORDER BY (boosted_until IS NOT NULL AND boosted_until > NOW()) DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTrendingIndexBots :many
WITH scored AS (
	SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS score
	FROM entity_votes
	WHERE target_type = 'bot' AND void = false AND created_at > now() - interval '7 days'
	GROUP BY target_id
)
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots
WHERE (type = 'approved' OR type = 'certified') AND bot_id IN (SELECT target_id FROM scored)
ORDER BY (SELECT score FROM scored WHERE scored.target_id = bots.bot_id) DESC
LIMIT $1 OFFSET $2;

-- name: CountBots :one
SELECT COUNT(*) FROM bots;

-- name: CountTrendingBots :one
SELECT COUNT(*) FROM bots
WHERE (type = 'approved' OR type = 'certified') AND bot_id IN (
	SELECT target_id FROM entity_votes
	WHERE target_type = 'bot' AND void = false AND created_at > now() - interval '7 days'
	GROUP BY target_id
);

-- name: GetCertifiedIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE type = 'certified' ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetPremiumIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE premium = true ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetMostViewedIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY clicks DESC LIMIT 9;

-- name: GetRecentlyAddedIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE type = 'approved' ORDER BY created_at DESC LIMIT 9;

-- name: GetTopVotedIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE type = 'approved' OR type = 'certified' ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetFeaturedIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE (type = 'approved' OR type = 'certified') AND featured_until IS NOT NULL AND featured_until > NOW() ORDER BY featured_until DESC LIMIT 9;

-- name: GetSpotlightIndexBots :many
SELECT bot_id, short, type, vanity_ref, approximate_votes, shards, library, invite_clicks, clicks, servers, nsfw, tags, premium, created_at, self_status, last_stats_post, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until
FROM bots WHERE (type = 'approved' OR type = 'certified') AND spotlighted_until IS NOT NULL AND spotlighted_until > NOW() ORDER BY spotlighted_until DESC LIMIT 9;

-- name: GetBotCommands :many
SELECT id, name, description, usage, category, position, created_at, updated_at
FROM bot_commands
WHERE bot_id = $1
ORDER BY position ASC, created_at ASC;

-- name: DeleteBotCommands :exec
DELETE FROM bot_commands WHERE bot_id = $1;

-- name: InsertBotCommand :exec
INSERT INTO bot_commands (bot_id, name, description, usage, category, position) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetBotChangelogs :many
SELECT id, title, content, version, created_by, created_at
FROM bot_changelogs
WHERE bot_id = $1
ORDER BY created_at DESC;

-- name: InsertBotChangelog :one
INSERT INTO bot_changelogs (bot_id, title, content, version, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, title, content, version, created_by, created_at;

-- name: DeleteBotChangelog :execrows
DELETE FROM bot_changelogs WHERE id = $1 AND bot_id = $2;

-- name: DeleteBotByID :exec
DELETE FROM bots WHERE bot_id = $1;

-- name: InsertBot :exec
INSERT INTO bots (bot_id, client_id, short, long, prefix, invite, library, extra_links, tags, nsfw, approval_note, team_owner, servers, vanity_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: UpdateBotModerationResult :exec
UPDATE bots SET moderation_flagged = $2, moderation_categories = $3 WHERE bot_id = $1;

-- name: UpdateBotSettings :exec
UPDATE bots SET short = $1, long = $2, prefix = $3, invite = $4, library = $5, extra_links = $6, tags = $7, nsfw = $8, captcha_opt_out = $9, updated_at = NOW()
WHERE bot_id = $10;

-- name: GetBotTeamOwner :one
SELECT team_owner FROM bots WHERE bot_id = $1;

-- name: UpdateBotTeamOwner :exec
UPDATE bots SET team_owner = $1, owner = NULL WHERE bot_id = $2;

-- name: UpdateBotLastStatsPost :exec
UPDATE bots SET last_stats_post = NOW() WHERE bot_id = $1;

-- name: UpdateBotServers :exec
UPDATE bots SET servers = $1 WHERE bot_id = $2;

-- name: UpdateBotShards :exec
UPDATE bots SET shards = $1 WHERE bot_id = $2;

-- name: UpdateBotUsers :exec
UPDATE bots SET users = $1 WHERE bot_id = $2;

-- name: UpdateBotShardList :exec
UPDATE bots SET shard_list = $1 WHERE bot_id = $2;

-- name: UpdateBotSelfStatus :exec
UPDATE bots SET self_status = $1 WHERE bot_id = $2;
