-- name: UpsertEntityAssetVersion :exec
INSERT INTO entity_assets (target_type, target_id, kind, version)
VALUES ($1, $2, $3, $4)
ON CONFLICT (target_type, target_id, kind)
DO UPDATE SET version = EXCLUDED.version, updated_at = NOW();

-- name: GetEntityAssetVersions :many
SELECT target_id, kind, version FROM entity_assets
WHERE target_type = sqlc.arg(target_type) AND target_id = ANY(sqlc.arg(target_ids)::text[]);
