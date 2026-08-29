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
