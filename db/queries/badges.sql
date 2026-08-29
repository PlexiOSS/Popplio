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
