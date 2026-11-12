-- name: GetReportStats :many
SELECT reason, status, COUNT(*) AS count FROM reports GROUP BY reason, status ORDER BY reason, status;

-- name: CountDailyReports :one
SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND created_at > NOW() - INTERVAL '24 hours';

-- name: InsertReport :exec
INSERT INTO reports (target_type, target_id, reporter_id, reason, description) VALUES ($1, $2, $3, $4, $5);

-- name: InsertAutoReport :exec
INSERT INTO reports (target_type, target_id, reporter_id, reason, description) VALUES ($1, $2, $3, 'tos_violation', $4) ON CONFLICT DO NOTHING;

-- name: ListReports :many
SELECT id, target_type, target_id, reporter_id, reason, description, status, resolved_by, resolution_note, created_at, resolved_at
FROM reports
WHERE sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')
ORDER BY created_at DESC;

-- name: ResolveReportReturningReporter :one
UPDATE reports SET status = $1, resolved_by = $2, resolution_note = $3, resolved_at = NOW()
WHERE id = $4 AND status IN ('open', 'under_review')
RETURNING reporter_id;

-- Report-target existence/name lookups, one per reportable entity type
-- (popplio/reports.GetTargetInfo dispatches to the right one -- sqlc needs
-- a static table per query, same reasoning as db/queries/payments.sql).

-- name: BotExists :one
SELECT EXISTS(SELECT 1 FROM bots WHERE bot_id = $1);

-- name: GetServerName :one
SELECT name FROM servers WHERE server_id = $1;

-- name: GetPackName :one
SELECT name FROM packs WHERE url = $1;

-- name: GetTeamName :one
SELECT name FROM teams WHERE id = $1;

-- name: InsertSystemReport :execrows
INSERT INTO reports (target_type, target_id, reporter_id, reason, description) VALUES ($1, $2, $3, 'tos_violation', $4) ON CONFLICT DO NOTHING;

-- name: ListReportsPage :many
SELECT r.id, r.target_type, r.target_id, r.reporter_id, r.reason, r.description, r.status, r.resolved_by, r.resolution_note, r.created_at, r.resolved_at,
    COALESCE(CASE r.target_type
        WHEN 'bot' THEN (SELECT NULLIF(c.display_name, '') FROM internal_user_cache__discord c WHERE c.id = r.target_id)
        WHEN 'server' THEN (SELECT s.name FROM servers s WHERE s.server_id = r.target_id)
        WHEN 'pack' THEN (SELECT p.name FROM packs p WHERE p.url = r.target_id)
        WHEN 'team' THEN (SELECT t.name FROM teams t WHERE t.id::text = r.target_id)
    END, CASE WHEN r.target_type = 'bot' THEN (SELECT c.username FROM internal_user_cache__discord c WHERE c.id = r.target_id) END, '')::text AS target_name,
    COALESCE(CASE r.target_type
        WHEN 'bot' THEN (SELECT c.avatar FROM internal_user_cache__discord c WHERE c.id = r.target_id)
        WHEN 'server' THEN (SELECT s.avatar FROM servers s WHERE s.server_id = r.target_id)
    END, '')::text AS target_avatar,
    COUNT(*) OVER ()::int AS total
FROM reports r
WHERE sqlc.narg('status')::text IS NULL OR r.status = sqlc.narg('status')
ORDER BY r.created_at DESC
LIMIT sqlc.arg('limit')::int OFFSET sqlc.arg('offset')::int;
