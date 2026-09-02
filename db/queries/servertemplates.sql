-- name: InsertServerTemplate :one
INSERT INTO server_templates (code, name, short, tags, nsfw, owner, usage_count, channels, roles)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: CountServerTemplateByCode :one
SELECT EXISTS(SELECT 1 FROM server_templates WHERE code = $1);

-- name: CountServerTemplateByID :one
SELECT EXISTS(SELECT 1 FROM server_templates WHERE id = $1);

-- name: GetServerTemplateByID :one
-- The single-fetch (detail page) query only -- channels/roles are a
-- meaningful chunk of JSON per row, so the list query below deliberately
-- doesn't select them; a browse grid has no use for them.
SELECT id, code, name, short, tags, nsfw, owner, usage_count, created_at, updated_at, channels, roles
FROM server_templates WHERE id = $1;

-- name: GetServerTemplateOwner :one
SELECT owner FROM server_templates WHERE id = $1;

-- name: GetServerTemplatesPaged :many
SELECT id, code, name, short, tags, nsfw, owner, usage_count, created_at, updated_at
FROM server_templates
WHERE (cardinality(sqlc.arg('tags')::text[]) = 0 OR tags && sqlc.arg('tags')::text[])
AND (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountServerTemplatesFiltered :one
SELECT COUNT(*) FROM server_templates
WHERE (cardinality(sqlc.arg('tags')::text[]) = 0 OR tags && sqlc.arg('tags')::text[])
AND (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'));

-- name: DeleteServerTemplate :exec
DELETE FROM server_templates WHERE id = $1;
