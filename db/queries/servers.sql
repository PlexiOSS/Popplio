-- IndexServer is the reduced projection used for listings, search results,
-- and embedding a server inside other entities (packs, etc.) -- shared
-- across several packages, hence its own file even though "servers" the
-- full package hasn't been converted yet.

-- name: GetIndexServerByID :one
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE server_id = $1;

-- name: GetIndexServersByTeamMembership :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE team_owner IN (SELECT team_id FROM team_members WHERE user_id = $1);

-- name: GetIndexServersByTeamOwner :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE team_owner = $1;

-- name: GetServersDueForModerationScan :many
SELECT server_id, short, long
FROM servers
WHERE type IN ('approved', 'certified', 'pending')
AND (moderation_checked_at IS NULL OR moderation_checked_at < NOW() - INTERVAL '7 days')
ORDER BY moderation_checked_at ASC NULLS FIRST
LIMIT $1;

-- name: RecordServerModerationScan :exec
UPDATE servers SET moderation_flagged = $2, moderation_categories = $3, moderation_checked_at = NOW() WHERE server_id = $1;

-- name: GetServerCertStats :one
SELECT type, total_members, cardinality(unique_clicks) AS unique_clicks, approximate_votes, created_at
FROM servers
WHERE server_id = $1;

-- name: CertifyServer :exec
UPDATE servers SET type = 'certified' WHERE server_id = $1;

-- name: GetServerCaptchaOptOut :one
SELECT captcha_opt_out FROM servers WHERE server_id = $1;

-- name: GetServersWithEmojisShown :many
SELECT server_id FROM servers WHERE show_emojis = true;

-- name: UpdateServerEmojisStickers :exec
UPDATE servers SET emojis = $2, stickers = $3, emojis_synced_at = NOW() WHERE server_id = $1;

-- name: GetServerIDsAndStatsSelfManaged :many
SELECT server_id, stats_self_managed FROM servers;

-- name: UpdateServerAvatarAndNsfwStats :exec
UPDATE servers SET avatar = $2, discord_nsfw_level = $3, nsfw_channel_count = $4 WHERE server_id = $1;

-- name: UpdateServerAvatarMembersAndNsfw :exec
UPDATE servers SET avatar = $2, total_members = $3, online_members = $4, discord_nsfw_level = $5, nsfw_channel_count = $6 WHERE server_id = $1;

-- name: GetServerInviteEligibility :one
SELECT login_required_for_invite, blacklisted_users, invite, type, state FROM servers WHERE server_id = $1;

-- name: UpdateServerInvite :exec
UPDATE servers SET invite = $2 WHERE server_id = $1;

-- name: UpdateServerShortLong :exec
UPDATE servers SET short = $2, long = $3 WHERE server_id = $1;

-- name: GetServerVanityRef :one
SELECT vanity_ref FROM servers WHERE server_id = $1;

-- name: DeleteServerByID :exec
DELETE FROM servers WHERE server_id = $1;

-- name: GetServerTypeAndClaimedBy :one
SELECT type, claimed_by FROM servers WHERE server_id = $1;

-- name: UpdateServerClaim :exec
UPDATE servers SET last_claimed = NOW(), claimed_by = $1 WHERE server_id = $2;

-- name: UncertifyServer :exec
UPDATE servers SET type = 'approved' WHERE server_id = $1;

-- name: ClearServerFeaturedUntil :exec
UPDATE servers SET featured_until = NULL WHERE server_id = $1;

-- name: SetServerSpotlightedUntil :exec
UPDATE servers SET spotlighted_until = GREATEST(COALESCE(spotlighted_until, NOW()), NOW()) + make_interval(hours => sqlc.arg('hours')::int) WHERE server_id = sqlc.arg('server_id')::text;

-- name: ClearServerSpotlightedUntil :exec
UPDATE servers SET spotlighted_until = NULL WHERE server_id = $1;

-- name: RemoveServerPremium :exec
UPDATE servers SET premium = false WHERE server_id = $1;

-- name: GetServerReviewStatus :one
SELECT type, claimed_by, last_claimed FROM servers WHERE server_id = $1;

-- name: ApproveServer :exec
UPDATE servers SET type = 'approved', claimed_by = NULL WHERE server_id = $1;

-- name: DenyServer :exec
UPDATE servers SET type = 'denied', claimed_by = NULL WHERE server_id = $1;

-- name: UnverifyServer :exec
UPDATE servers SET type = 'pending', claimed_by = NULL WHERE server_id = $1;

-- name: SetServerVoteBanned :exec
UPDATE servers SET vote_banned = $2 WHERE server_id = $1;

-- name: GetServerTeamOwner :one
SELECT team_owner FROM servers WHERE server_id = $1;

-- name: GetServerNameAndShort :one
SELECT name, short FROM servers WHERE server_id = $1;

-- name: GetServerNameAndAvatar :one
SELECT name, avatar FROM servers WHERE server_id = $1;

-- name: GetServerType :one
SELECT type FROM servers WHERE server_id = $1;

-- name: GetRandomIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE (type = 'approved' OR type = 'certified') AND state = 'public'
ORDER BY RANDOM()
LIMIT 3;

-- name: GetFlatEmojis :many
SELECT s.server_id AS server_id, s.name AS server_name, s.avatar AS server_avatar,
	(e->>'id')::text AS id, (e->>'name')::text AS name, coalesce((e->>'animated')::boolean, false)::boolean AS animated, (e->>'url')::text AS url
FROM servers s
CROSS JOIN LATERAL jsonb_array_elements(s.emojis) AS e
WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'
ORDER BY e->>'name' ASC
LIMIT $1 OFFSET $2;

-- name: CountFlatEmojis :one
SELECT coalesce(SUM(jsonb_array_length(s.emojis)), 0)::bigint
FROM servers s
WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public';

-- name: GetFlatStickers :many
SELECT s.server_id AS server_id, s.name AS server_name, s.avatar AS server_avatar,
	(e->>'id')::text AS id, (e->>'name')::text AS name, (e->>'format')::text AS format, (e->>'url')::text AS url
FROM servers s
CROSS JOIN LATERAL jsonb_array_elements(s.stickers) AS e
WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public'
ORDER BY e->>'name' ASC
LIMIT $1 OFFSET $2;

-- name: CountFlatStickers :one
SELECT coalesce(SUM(jsonb_array_length(s.stickers)), 0)::bigint
FROM servers s
WHERE s.show_emojis = true AND (s.type = 'approved' OR s.type = 'certified') AND s.state = 'public';

-- name: GetServerEmojiPreviews :many
SELECT server_id, name, avatar, emojis, stickers
FROM servers
WHERE show_emojis = true AND (type = 'approved' OR type = 'certified') AND state = 'public'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountServerEmojiPreviews :one
SELECT COUNT(*) FROM servers WHERE show_emojis = true AND (type = 'approved' OR type = 'certified') AND state = 'public';

-- name: GetIndexServersPaged :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE (type = 'approved' OR type = 'certified') AND state = 'public'
ORDER BY (boosted_until IS NOT NULL AND boosted_until > NOW()) DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTrendingIndexServers :many
WITH scored AS (
	SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS score
	FROM entity_votes
	WHERE target_type = 'server' AND void = false AND created_at > now() - interval '7 days'
	GROUP BY target_id
)
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers
WHERE (type = 'approved' OR type = 'certified') AND state = 'public' AND server_id IN (SELECT target_id FROM scored)
ORDER BY (SELECT score FROM scored WHERE scored.target_id = servers.server_id) DESC
LIMIT $1 OFFSET $2;

-- name: CountServers :one
SELECT COUNT(*) FROM servers;

-- name: CountServersByType :many
SELECT type AS method, COUNT(*) FROM servers GROUP BY type;

-- Panel server review queue and search (arcadia/panel ops_queue.go, ops_search.go).

-- name: ServerQueuePending :many
SELECT server_id, name, avatar, total_members, online_members, short, type, approval_note,
       approximate_votes, invite_clicks, clicks, nsfw, discord_nsfw_level, nsfw_channel_count,
       tags, premium, claimed_by, last_claimed, moderation_flagged, moderation_categories
FROM servers WHERE type = 'pending' OR type = 'claimed' ORDER BY created_at;

-- name: SearchServersQueue :many
SELECT server_id, name, avatar, total_members, online_members, short, type, approval_note,
       approximate_votes, invite_clicks, clicks, nsfw, discord_nsfw_level, nsfw_channel_count,
       tags, premium, claimed_by, last_claimed, moderation_flagged, moderation_categories
FROM servers WHERE server_id = sqlc.arg('query')::text OR name ILIKE sqlc.arg('pattern')::text ORDER BY created_at;

-- name: CountTrendingServers :one
SELECT COUNT(*) FROM servers
WHERE (type = 'approved' OR type = 'certified') AND state = 'public' AND server_id IN (
	SELECT target_id FROM entity_votes
	WHERE target_type = 'server' AND void = false AND created_at > now() - interval '7 days'
	GROUP BY target_id
);

-- name: GetCertifiedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND type = 'certified' ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetPremiumIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND premium = true ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetMostViewedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') ORDER BY clicks DESC LIMIT 9;

-- name: GetRecentlyAddedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND type = 'approved' ORDER BY created_at DESC LIMIT 9;

-- name: GetTopVotedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') ORDER BY approximate_votes DESC LIMIT 9;

-- name: GetFeaturedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') AND featured_until IS NOT NULL AND featured_until > NOW() ORDER BY featured_until DESC LIMIT 9;

-- name: GetSpotlightIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') AND spotlighted_until IS NOT NULL AND spotlighted_until > NOW() ORDER BY spotlighted_until DESC LIMIT 9;

-- name: GetServerByID :one
SELECT server_id, name, avatar, total_members, online_members, short, type, approval_note, state, tags, vanity_ref, extra_links, team_owner, invite_clicks, clicks, nsfw, approximate_votes, vote_banned, premium, start_premium_period, premium_period_length, captcha_opt_out, created_at, claimed_by, last_claimed, login_required_for_invite, show_emojis, emojis, stickers, emojis_synced_at, supporter_badge, boosted_until, featured_until, spotlighted_until, vote_blitz_until, discord_nsfw_level, nsfw_channel_count, moderation_flagged, moderation_categories
FROM servers
WHERE server_id = $1;

-- name: GetServerUniqueClicksCount :one
SELECT cardinality(unique_clicks) FROM servers WHERE server_id = $1;

-- name: GetServerLongDescription :one
SELECT long FROM servers WHERE server_id = $1;

-- name: UpdateServerClicks :exec
UPDATE servers SET clicks = clicks + 1 WHERE server_id = $1;

-- name: CheckServerHasUniqueClick :one
SELECT sqlc.arg('hashed_ip')::text = ANY(unique_clicks) FROM servers WHERE server_id = sqlc.arg('server_id')::text;

-- name: AppendServerUniqueClick :exec
UPDATE servers SET unique_clicks = array_append(unique_clicks, $1) WHERE server_id = $2;

-- name: UpdateServerInviteClicks :exec
UPDATE servers SET invite_clicks = invite_clicks + 1 WHERE server_id = $1;

-- name: InsertServer :exec
INSERT INTO servers (invite, short, long, extra_links, tags, nsfw, team_owner, server_id, name, avatar, total_members, online_members, vanity_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateServerModerationResult :exec
UPDATE servers SET moderation_flagged = $2, moderation_categories = $3 WHERE server_id = $1;

-- name: UpdateServerSettings :exec
UPDATE servers SET short = $1, long = $2, extra_links = $3, state = $4, tags = $5, nsfw = $6, captcha_opt_out = $7, login_required_for_invite = $8, show_emojis = $9
WHERE server_id = $10;

-- name: UpdateServerStatsMeta :exec
UPDATE servers SET last_stats_post = NOW(), stats_self_managed = true WHERE server_id = $1;

-- name: UpdateServerTotalMembers :exec
UPDATE servers SET total_members = $1 WHERE server_id = $2;

-- name: UpdateServerOnlineMembers :exec
UPDATE servers SET online_members = $1 WHERE server_id = $2;
