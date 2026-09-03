-- name: InsertTheme :exec
INSERT INTO themes (id, name, primary_color, secondary_color, tags, owner)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetThemeByID :one
SELECT id, owner, name, primary_color, secondary_color, tags, created_at
FROM themes WHERE id = $1;

-- name: GetThemeOwner :one
SELECT owner FROM themes WHERE id = $1;

-- name: DeleteTheme :exec
DELETE FROM themes WHERE id = $1;

-- name: GetAllThemesPaged :many
SELECT id, owner, name, primary_color, secondary_color, tags, created_at
FROM themes
WHERE (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountThemes :one
SELECT COUNT(*) FROM themes
WHERE (sqlc.arg('owner')::text = '' OR owner = sqlc.arg('owner'));
