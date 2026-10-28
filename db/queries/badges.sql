-- name: GetEntityBadges :many
SELECT eb.reason, eb.awarded_by, eb.created_at,
       b.id AS badge_id, b.name, b.description, b.icon, b.color, b.target_types
FROM entity_badges eb
INNER JOIN badges b ON b.id = eb.badge_id
WHERE eb.target_type = $1 AND eb.target_id = $2
ORDER BY eb.created_at ASC;

-- The badge catalog itself (panel ops_badges.go).

-- name: ListBadges :many
SELECT id, name, description, icon, color, target_types, created_at, created_by, last_updated, updated_by, auto_rule, linked_role
FROM badges ORDER BY created_at DESC;

-- name: InsertBadge :exec
INSERT INTO badges (id, name, description, icon, color, target_types, created_by, updated_by, auto_rule, linked_role) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, $9);

-- name: CountBadgeByID :one
SELECT EXISTS(SELECT 1 FROM badges WHERE id = $1);

-- name: UpdateBadge :exec
UPDATE badges SET name = $1, description = $2, icon = $3, color = $4, target_types = $5, last_updated = NOW(), updated_by = $6, auto_rule = $8, linked_role = $9 WHERE id = $7;

-- name: DeleteBadge :exec
DELETE FROM badges WHERE id = $1;

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

-- name: ListAutoBadges :many
SELECT id, target_types, auto_rule FROM badges WHERE auto_rule IS NOT NULL ORDER BY id;

-- name: RevokeOrphanSystemBadges :execrows
DELETE FROM entity_badges eb
USING badges b
WHERE eb.badge_id = b.id
  AND eb.awarded_by = 'system'
  AND (b.auto_rule IS NULL OR NOT (eb.target_type = ANY(b.target_types)));

-- name: ListLinkedRoleBadges :many
SELECT id, name, description FROM badges WHERE linked_role = true ORDER BY created_at, id;

-- name: CountOtherLinkedRoleBadges :one
SELECT COUNT(*) FROM badges WHERE linked_role = true AND id <> $1;

-- name: GetUserLinkedRoleBadgeIDs :many
SELECT eb.badge_id FROM entity_badges eb
INNER JOIN badges b ON b.id = eb.badge_id
WHERE eb.target_type = 'user' AND eb.target_id = $1 AND b.linked_role = true;

-- name: UpsertUserRoleConnection :exec
INSERT INTO user_role_connections (user_id, access_token, refresh_token, expires_at, last_metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET access_token = $2, refresh_token = $3, expires_at = $4, last_metadata = $5, updated_at = NOW();

-- name: ListUserRoleConnections :many
SELECT user_id, access_token, refresh_token, expires_at, last_metadata FROM user_role_connections;

-- name: GetUserRoleConnection :one
SELECT user_id, access_token, refresh_token, expires_at, last_metadata FROM user_role_connections WHERE user_id = $1;

-- name: UpdateUserRoleConnectionTokens :exec
UPDATE user_role_connections SET access_token = $2, refresh_token = $3, expires_at = $4, updated_at = NOW() WHERE user_id = $1;

-- name: UpdateUserRoleConnectionMetadata :exec
UPDATE user_role_connections SET last_metadata = $2, updated_at = NOW() WHERE user_id = $1;

-- name: DeleteUserRoleConnection :exec
DELETE FROM user_role_connections WHERE user_id = $1;

-- name: ListPublicBadges :many
SELECT b.id, b.name, b.description, b.icon, b.color, b.target_types, b.auto_rule, b.linked_role,
       (SELECT COUNT(*) FROM entity_badges eb WHERE eb.badge_id = b.id)::int AS holders
FROM badges b
ORDER BY (b.auto_rule IS NULL), b.name;
