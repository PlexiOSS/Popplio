-- name: GetPageStyle :one
SELECT target_type, target_id, gradient, accent, image_version, pending_image_version, pending_since, rejected_reason, dim, blur, image_position, fixed, updated_at
FROM entity_page_styles WHERE target_type = $1 AND target_id = $2;

-- name: UpsertPageStyleSettings :exec
INSERT INTO entity_page_styles (target_type, target_id, gradient, accent, dim, blur, image_position, fixed)
VALUES (sqlc.arg(target_type), sqlc.arg(target_id), sqlc.narg(gradient), sqlc.narg(accent), sqlc.arg(dim), sqlc.arg(blur), sqlc.arg(image_position), sqlc.arg(fixed))
ON CONFLICT (target_type, target_id) DO UPDATE SET
    gradient = EXCLUDED.gradient,
    accent = EXCLUDED.accent,
    dim = EXCLUDED.dim,
    blur = EXCLUDED.blur,
    image_position = EXCLUDED.image_position,
    fixed = EXCLUDED.fixed,
    updated_at = NOW();

-- name: SetPendingPageImage :exec
INSERT INTO entity_page_styles (target_type, target_id, pending_image_version, pending_since)
VALUES (sqlc.arg(target_type), sqlc.arg(target_id), sqlc.arg(version), NOW())
ON CONFLICT (target_type, target_id) DO UPDATE SET
    pending_image_version = EXCLUDED.pending_image_version,
    pending_since = NOW(),
    rejected_reason = NULL,
    updated_at = NOW();

-- name: RemovePageImage :exec
UPDATE entity_page_styles SET image_version = NULL, pending_image_version = NULL, pending_since = NULL, rejected_reason = NULL, updated_at = NOW()
WHERE target_type = $1 AND target_id = $2;

-- name: ListPendingPageImages :many
SELECT target_type, target_id, pending_image_version::text AS version, pending_since::timestamptz AS pending_since, image_version, dim, blur, image_position
FROM entity_page_styles
WHERE pending_image_version IS NOT NULL
ORDER BY pending_since ASC;

-- name: CountPendingPageImages :one
SELECT COUNT(*) FROM entity_page_styles WHERE pending_image_version IS NOT NULL;

-- name: ApprovePageImage :execrows
UPDATE entity_page_styles SET
    image_version = pending_image_version,
    pending_image_version = NULL,
    pending_since = NULL,
    rejected_reason = NULL,
    reviewed_by = sqlc.arg(staff_id),
    reviewed_at = NOW(),
    updated_at = NOW()
WHERE target_type = sqlc.arg(target_type) AND target_id = sqlc.arg(target_id) AND pending_image_version = sqlc.arg(version);

-- name: RejectPageImage :execrows
UPDATE entity_page_styles SET
    pending_image_version = NULL,
    pending_since = NULL,
    rejected_reason = sqlc.arg(reason),
    reviewed_by = sqlc.arg(staff_id),
    reviewed_at = NOW(),
    updated_at = NOW()
WHERE target_type = sqlc.arg(target_type) AND target_id = sqlc.arg(target_id) AND pending_image_version = sqlc.arg(version);
