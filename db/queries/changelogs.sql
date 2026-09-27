-- name: GetChangelogList :many
SELECT itag, project, version, added, updated, fixed, security, removed, extra_description, prerelease, created_by, created_at
FROM changelogs
WHERE published = true AND (sqlc.narg('project')::text IS NULL OR project = sqlc.narg('project'))
ORDER BY created_at DESC;

-- Panel changelog CRUD (arcadia/panel ops_content.go) sees every entry,
-- published or not, which is why it can't reuse GetChangelogList above.

-- name: ListChangelogEntries :many
SELECT itag, project, version, added, updated, fixed, security, removed, extra_description, prerelease, published, created_by, created_at
FROM changelogs ORDER BY created_at DESC;

-- name: InsertChangelogEntry :exec
INSERT INTO changelogs (project, version, added, updated, fixed, security, removed, extra_description, prerelease, published, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE(sqlc.narg('created_at')::timestamptz, NOW()));

-- name: CountChangelogByItag :one
SELECT EXISTS(SELECT 1 FROM changelogs WHERE itag = $1);

-- name: GetChangelogPublishedByItag :one
SELECT published FROM changelogs WHERE itag = $1;

-- name: UpdateChangelogEntry :exec
UPDATE changelogs SET project = $2, version = $3, added = $4, updated = $5, fixed = $6, security = $7, removed = $8, extra_description = $9, prerelease = $10, published = $11, created_at = COALESCE(sqlc.narg('created_at')::timestamptz, created_at)
WHERE itag = $1;

-- name: DeleteChangelogByItag :exec
DELETE FROM changelogs WHERE itag = $1;
