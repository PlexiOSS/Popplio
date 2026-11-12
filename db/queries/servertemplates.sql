-- name: InsertServerTemplate :one
INSERT INTO server_templates (code, name, short, tags, nsfw, owner, usage_count, channels, roles, source_guild_id, source_updated_at, out_of_date, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
RETURNING id;

-- name: CountServerTemplateByCode :one
SELECT EXISTS(SELECT 1 FROM server_templates WHERE code = $1);

-- name: CountServerTemplateByID :one
SELECT EXISTS(SELECT 1 FROM server_templates WHERE id = $1);

-- name: GetServerTemplateByID :one
-- The single-fetch (detail page) query only -- channels/roles are a
-- meaningful chunk of JSON per row, so the list query below deliberately
-- doesn't select them; a browse grid has no use for them.
SELECT id, code, name, short, tags, nsfw, owner, usage_count, created_at, updated_at, channels, roles,
    icon_version, source_updated_at, out_of_date, unavailable, synced_at
FROM server_templates WHERE id = $1;

-- name: GetServerTemplateOwner :one
SELECT owner FROM server_templates WHERE id = $1;

-- name: GetServerTemplateCode :one
SELECT owner, code FROM server_templates WHERE id = $1;

-- name: GetServerTemplatesPaged :many
SELECT id, code, name, short, tags, nsfw, owner, usage_count, created_at, updated_at,
    icon_version, source_updated_at, out_of_date, unavailable, synced_at,
    (SELECT COUNT(*) FROM jsonb_array_elements(channels) c WHERE (c->>'type')::int <> 4)::int AS channel_count,
    jsonb_array_length(roles)::int AS role_count
FROM server_templates
WHERE (cardinality(sqlc.arg('tags')::text[]) = 0 OR tags && sqlc.arg('tags')::text[])
AND (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'))
AND (sqlc.arg('owner')::text <> '' OR NOT unavailable)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountServerTemplatesFiltered :one
SELECT COUNT(*) FROM server_templates
WHERE (cardinality(sqlc.arg('tags')::text[]) = 0 OR tags && sqlc.arg('tags')::text[])
AND (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'))
AND (sqlc.arg('owner')::text <> '' OR NOT unavailable);

-- name: DeleteServerTemplate :exec
DELETE FROM server_templates WHERE id = $1;

-- name: UpdateServerTemplate :exec
UPDATE server_templates
SET short = $2, tags = $3, nsfw = $4, updated_at = now()
WHERE id = $1;

-- name: SyncServerTemplate :exec
UPDATE server_templates
SET name = $2, usage_count = $3, channels = $4, roles = $5, source_guild_id = $6,
    source_updated_at = $7, out_of_date = $8, unavailable = false, synced_at = now()
WHERE id = $1;

-- name: MarkServerTemplateUnavailable :exec
UPDATE server_templates SET unavailable = true, synced_at = now() WHERE id = $1;

-- name: SetServerTemplateIcon :exec
UPDATE server_templates SET icon_version = $2, updated_at = now() WHERE id = $1;

-- name: ListServerTemplatesToSync :many
SELECT id, code FROM server_templates
WHERE synced_at IS NULL OR synced_at < now() - interval '1 day'
ORDER BY synced_at NULLS FIRST
LIMIT $1;
