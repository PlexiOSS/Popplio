-- name: GetEntityBadges :many
SELECT eb.reason, eb.awarded_by, eb.created_at,
       b.id AS badge_id, b.name, b.description, b.icon, b.color, b.target_types
FROM entity_badges eb
INNER JOIN badges b ON b.id = eb.badge_id
WHERE eb.target_type = $1 AND eb.target_id = $2
ORDER BY eb.created_at ASC;

-- The badge catalog itself (panel ops_badges.go).

-- name: ListBadges :many
SELECT id, name, description, icon, color, target_types, created_at, created_by, last_updated, updated_by
FROM badges ORDER BY created_at DESC;

-- name: InsertBadge :exec
INSERT INTO badges (id, name, description, icon, color, target_types, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $7);

-- name: CountBadgeByID :one
SELECT EXISTS(SELECT 1 FROM badges WHERE id = $1);

-- name: UpdateBadge :exec
UPDATE badges SET name = $1, description = $2, icon = $3, color = $4, target_types = $5, last_updated = NOW(), updated_by = $6 WHERE id = $7;

-- name: DeleteBadge :exec
DELETE FROM badges WHERE id = $1;

-- name: GetPartnerBotIDs :many
SELECT DISTINCT b.bot_id FROM partners p
INNER JOIN bots b ON b.bot_id = p.bot_id
WHERE b.type = 'approved' OR b.type = 'certified';

-- name: GetVeteranBotIDs :many
SELECT bot_id FROM bots
WHERE (type = 'approved' OR type = 'certified') AND created_at < NOW() - INTERVAL '2 years';

-- name: GetVeteranServerIDs :many
SELECT server_id FROM servers
WHERE (type = 'approved' OR type = 'certified') AND created_at < NOW() - INTERVAL '2 years';

-- name: AwardSystemBadge :execrows
INSERT INTO entity_badges (target_type, target_id, badge_id, reason, awarded_by)
SELECT sqlc.arg(target_type)::text, t, sqlc.arg(badge_id)::text, sqlc.arg(reason)::text, 'system'
FROM unnest(sqlc.arg(target_ids)::text[]) AS t
WHERE EXISTS (SELECT 1 FROM badges WHERE id = sqlc.arg(badge_id)::text)
ON CONFLICT (target_type, target_id, badge_id) DO NOTHING;

-- name: RevokeSystemBadge :execrows
DELETE FROM entity_badges
WHERE target_type = sqlc.arg(target_type)::text
  AND badge_id = sqlc.arg(badge_id)::text
  AND awarded_by = 'system'
  AND NOT (target_id = ANY(sqlc.arg(target_ids)::text[]));
