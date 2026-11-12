-- name: GetListStats :one
-- Folded from 17 separate COUNT(*) round-trips into one query with
-- sub-selects -- same data, one round trip instead of seventeen.
SELECT
    (SELECT COUNT(*) FROM bots) AS total_bots,
    (SELECT COUNT(*) FROM bots WHERE type = 'approved') AS total_approved_bots,
    (SELECT COUNT(*) FROM bots WHERE type = 'certified') AS total_certified_bots,
    (SELECT COUNT(*) FROM bots WHERE type = 'pending') AS total_pending_bots,
    (SELECT COUNT(*) FROM bots WHERE type = 'denied') AS total_denied_bots,
    (SELECT COUNT(*) FROM staff_members) AS total_staff,
    (SELECT COUNT(*) FROM users) AS total_users,
    -- COALESCE guards against SUM() returning NULL when bots is empty --
    -- the original raw-SQL version scanned straight into a non-nullable
    -- int64 and would have panicked on an empty table. Found via live
    -- testing against the (empty) test database.
    (SELECT COALESCE(SUM(approximate_votes), 0)::bigint FROM bots) AS total_votes,
    (SELECT COUNT(*) FROM packs) AS total_packs,
    (SELECT COUNT(*) FROM tickets) AS total_tickets,
    (SELECT COUNT(*) FROM users WHERE banned = true) AS total_banned_users,
    (SELECT COUNT(*) FROM bots WHERE vote_banned = true) AS total_vote_banned_bots,
    (SELECT COUNT(*) FROM servers) AS total_servers,
    (SELECT COUNT(*) FROM servers WHERE type = 'approved') AS total_approved_servers,
    (SELECT COUNT(*) FROM servers WHERE type = 'certified') AS total_certified_servers,
    (SELECT COUNT(*) FROM servers WHERE type = 'pending') AS total_pending_servers,
    (SELECT COUNT(*) FROM servers WHERE type = 'denied') AS total_denied_servers,
    (SELECT COUNT(*) FROM servers WHERE vote_banned = true) AS total_vote_banned_servers;

-- name: GetPartners :many
SELECT id, name, short, links, type, created_at, user_id, bot_id FROM partners ORDER BY created_at DESC;

-- name: GetPartnerTypes :many
SELECT id, name, short, icon, created_at FROM partner_types ORDER BY created_at DESC;

-- Shared by get_rss_feed and get_sitemap.

-- name: GetNewBotIDs :many
SELECT bot_id FROM bots WHERE (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetCertifiedBotIDs :many
SELECT bot_id FROM bots WHERE type = 'certified' ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetPremiumBotIDs :many
SELECT bot_id FROM bots WHERE premium = true AND (type = 'approved' OR type = 'certified') ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetStaffTemplates :many
SELECT id, name, emoji, tags, description, type, entity_type, created_at
FROM staff_templates
WHERE sqlc.narg('entity_type')::text IS NULL OR entity_type = sqlc.narg('entity_type')
ORDER BY created_at DESC;

-- name: GetStaffTemplateTypes :many
SELECT id, name, icon, short FROM staff_template_types ORDER BY created_at DESC;

-- name: GetListTags :many
WITH t AS (
    SELECT lower(btrim(tag)) AS key, btrim(tag) AS label, 'bot' AS kind
    FROM bots, unnest(tags) AS tag
    WHERE bots.type = 'approved' OR bots.type = 'certified'
    UNION ALL
    SELECT lower(btrim(tag)), btrim(tag), 'server'
    FROM servers, unnest(tags) AS tag
    WHERE (servers.type = 'approved' OR servers.type = 'certified') AND servers.state = 'public'
)
SELECT
    mode() WITHIN GROUP (ORDER BY label)::text AS label,
    (COUNT(*) FILTER (WHERE kind = 'bot'))::bigint AS bots,
    (COUNT(*) FILTER (WHERE kind = 'server'))::bigint AS servers
FROM t
WHERE key <> ''
GROUP BY key
ORDER BY COUNT(*) DESC, key;

-- name: GetStatsHistoryTotals :one
SELECT
    (SELECT COUNT(*) FROM entity_votes WHERE upvote)::bigint AS votes_all_time,
    (SELECT COUNT(*) FROM entity_votes WHERE upvote AND NOT void)::bigint AS votes_this_month,
    (SELECT COUNT(DISTINCT author) FROM entity_votes WHERE upvote)::bigint AS voters_all_time,
    (SELECT MIN(created_at) FROM entity_votes)::timestamptz AS votes_since,
    (SELECT COUNT(*) FROM rpc_logs WHERE method = 'Approve' AND state = 'success')::bigint AS approvals,
    (SELECT COUNT(*) FROM rpc_logs WHERE method = 'Deny' AND state = 'success')::bigint AS denials,
    (SELECT MIN(created_at) FROM rpc_logs WHERE method IN ('Approve', 'Deny') AND state = 'success')::timestamptz AS reviews_since,
    (SELECT MIN(created_at) FROM users)::timestamptz AS users_since;

-- name: GetMonthlyStatsHistory :many
WITH months AS (
    SELECT generate_series(
        date_trunc('month', LEAST(
            COALESCE((SELECT MIN(created_at) FROM bots), NOW()),
            COALESCE((SELECT MIN(created_at)::timestamptz FROM entity_votes), NOW())
        )),
        date_trunc('month', NOW()),
        INTERVAL '1 month'
    ) AS month
),
v AS (
    SELECT date_trunc('month', created_at::timestamptz) AS month,
        COUNT(*) FILTER (WHERE upvote) AS votes,
        COUNT(DISTINCT author) FILTER (WHERE upvote) AS voters
    FROM entity_votes GROUP BY 1
),
r AS (
    SELECT date_trunc('month', created_at) AS month,
        COUNT(*) FILTER (WHERE method = 'Approve') AS approved,
        COUNT(*) FILTER (WHERE method = 'Deny') AS denied
    FROM rpc_logs WHERE method IN ('Approve', 'Deny') AND state = 'success' GROUP BY 1
),
b AS (SELECT date_trunc('month', created_at) AS month, COUNT(*) AS n FROM bots GROUP BY 1),
s AS (SELECT date_trunc('month', created_at) AS month, COUNT(*) AS n FROM servers GROUP BY 1),
u AS (SELECT date_trunc('month', created_at) AS month, COUNT(*) AS n FROM users GROUP BY 1)
SELECT
    to_char(m.month, 'YYYY-MM')::text AS month,
    COALESCE(v.votes, 0)::bigint AS votes,
    COALESCE(v.voters, 0)::bigint AS voters,
    COALESCE(r.approved, 0)::bigint AS approved,
    COALESCE(r.denied, 0)::bigint AS denied,
    COALESCE(b.n, 0)::bigint AS new_bots,
    COALESCE(s.n, 0)::bigint AS new_servers,
    COALESCE(u.n, 0)::bigint AS new_users
FROM months m
LEFT JOIN v ON v.month = m.month
LEFT JOIN r ON r.month = m.month
LEFT JOIN b ON b.month = m.month
LEFT JOIN s ON s.month = m.month
LEFT JOIN u ON u.month = m.month
ORDER BY m.month;

-- name: GetBotStatusCounts :one
SELECT
    (SELECT COUNT(*) FROM bots WHERE type IN ('approved', 'certified'))::bigint AS listed_bots,
    (SELECT COUNT(*) FROM servers WHERE type IN ('approved', 'certified'))::bigint AS listed_servers,
    (SELECT COUNT(*) FROM bots WHERE type = 'pending')::bigint AS pending_bots,
    (SELECT COUNT(*) FROM servers WHERE type = 'pending')::bigint AS pending_servers;
