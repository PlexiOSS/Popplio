-- name: BumpServer :one
UPDATE servers SET bumped_at = NOW()
WHERE server_id = $1
  AND state = 'public'
  AND (type = 'approved' OR type = 'certified')
  AND (bumped_at IS NULL OR bumped_at <= NOW() - INTERVAL '2 hours')
RETURNING bumped_at;

-- name: GetServerBumpState :one
SELECT name, state, type, bumped_at FROM servers WHERE server_id = $1;

-- name: InsertServerBump :exec
INSERT INTO server_bumps (server_id, user_id) VALUES ($1, $2);

-- name: GetServerTopBumpers :many
SELECT user_id, COUNT(*)::int AS bumps
FROM server_bumps
WHERE server_id = $1 AND created_at > NOW() - INTERVAL '30 days'
GROUP BY user_id
ORDER BY bumps DESC, MAX(created_at) DESC
LIMIT sqlc.arg('lim');

-- name: CountServerBumpsSince :one
SELECT COUNT(*)::int FROM server_bumps WHERE server_id = $1 AND created_at > sqlc.arg('since');

-- name: GetRecentlyBumpedIndexServers :many
SELECT server_id, name, avatar, total_members, online_members, short, type, state, vanity_ref, approximate_votes, invite_clicks, clicks, nsfw, tags, premium, supporter_badge, boosted_until, featured_until, spotlighted_until
FROM servers WHERE state = 'public' AND (type = 'approved' OR type = 'certified') AND bumped_at IS NOT NULL ORDER BY bumped_at DESC LIMIT 9;
