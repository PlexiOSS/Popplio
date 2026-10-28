-- name: SnapshotBotStats :execrows
INSERT INTO entity_stats_daily (target_type, target_id, day, clicks, invite_clicks, servers, users)
SELECT 'bot', bot_id, (NOW() AT TIME ZONE 'UTC')::date, clicks, invite_clicks, servers, users
FROM bots WHERE type = 'approved' OR type = 'certified'
ON CONFLICT (target_type, target_id, day) DO UPDATE SET
    clicks = EXCLUDED.clicks,
    invite_clicks = EXCLUDED.invite_clicks,
    servers = EXCLUDED.servers,
    users = EXCLUDED.users,
    updated_at = NOW();

-- name: SnapshotServerStats :execrows
INSERT INTO entity_stats_daily (target_type, target_id, day, clicks, invite_clicks, total_members, online_members)
SELECT 'server', server_id, (NOW() AT TIME ZONE 'UTC')::date, clicks, invite_clicks, total_members, online_members
FROM servers WHERE type = 'approved' OR type = 'certified'
ON CONFLICT (target_type, target_id, day) DO UPDATE SET
    clicks = EXCLUDED.clicks,
    invite_clicks = EXCLUDED.invite_clicks,
    total_members = EXCLUDED.total_members,
    online_members = EXCLUDED.online_members,
    updated_at = NOW();

-- name: PruneEntityStats :execrows
DELETE FROM entity_stats_daily WHERE day < (NOW() AT TIME ZONE 'UTC')::date - 400;

-- name: GetEntityStatsSnapshots :many
SELECT day, clicks, invite_clicks, servers, users, total_members, online_members
FROM entity_stats_daily
WHERE target_type = $1 AND target_id = $2 AND day >= sqlc.arg('since')::date
ORDER BY day;

-- name: GetEntityDailyVotes :many
SELECT created_at::date AS day, COUNT(*)::int AS votes, COUNT(*) FILTER (WHERE auto)::int AS auto_votes
FROM entity_votes
WHERE target_type = $1 AND target_id = $2 AND upvote
  AND void_reason IS DISTINCT FROM 'Votes (single entity) reset'
  AND created_at >= sqlc.arg('since')::date
GROUP BY 1
ORDER BY 1;
