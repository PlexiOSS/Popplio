-- name: InsertAuditEvent :exec
INSERT INTO audit_events (source, actor_type, actor_id, action, target_type, target_id, status, ip, data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListAuditEvents :many
SELECT id, created_at, source, actor_type, actor_id, action, target_type, target_id, status, ip, data
FROM audit_events
WHERE (sqlc.narg(before)::bigint IS NULL OR id < sqlc.narg(before)::bigint)
  AND (sqlc.narg(source)::text IS NULL OR source = sqlc.narg(source)::text)
  AND (sqlc.narg(actor_id)::text IS NULL OR actor_id = sqlc.narg(actor_id)::text)
  AND (sqlc.narg(target_id)::text IS NULL OR target_id = sqlc.narg(target_id)::text)
  AND (sqlc.narg(action_prefix)::text IS NULL OR action LIKE sqlc.narg(action_prefix)::text || '%')
  AND (sqlc.narg(failed_only)::bool IS NULL OR NOT sqlc.narg(failed_only)::bool OR status >= 400)
ORDER BY id DESC
LIMIT sqlc.arg(lim)::int;

-- name: DeleteAuditEventsBefore :execrows
DELETE FROM audit_events WHERE created_at < $1;

-- name: ResolveAuditNames :many
SELECT id, COALESCE(NULLIF(display_name, ''), username)::text AS name, avatar::text AS avatar
FROM internal_user_cache__discord WHERE id = ANY(sqlc.arg(ids)::text[])
UNION ALL
SELECT server_id, name, avatar FROM servers WHERE server_id = ANY(sqlc.arg(ids)::text[])
UNION ALL
SELECT id::text, name, ''::text FROM teams WHERE id::text = ANY(sqlc.arg(ids)::text[]);
