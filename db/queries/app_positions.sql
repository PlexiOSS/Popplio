-- name: ListAppPositions :many
SELECT id, name, info, summary, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, help_url, help_label, behaviour, sort_order, created_at, updated_at
FROM app_positions
ORDER BY sort_order, created_at;

-- name: GetAppPosition :one
SELECT id, name, info, summary, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, help_url, help_label, behaviour, sort_order, created_at, updated_at
FROM app_positions
WHERE id = $1;

-- name: InsertAppPosition :exec
INSERT INTO app_positions (id, name, info, summary, tags, questions, hidden, closed, requires_main_server, cooldown_secs, approval_note, help_url, help_label, sort_order)
VALUES (
	sqlc.arg(id), sqlc.arg(name), sqlc.arg(info), sqlc.arg(summary), sqlc.arg(tags), sqlc.arg(questions), sqlc.arg(hidden), sqlc.arg(closed),
	sqlc.arg(requires_main_server), sqlc.arg(cooldown_secs), sqlc.arg(approval_note), sqlc.narg(help_url), sqlc.narg(help_label),
	COALESCE((SELECT MAX(sort_order) + 1 FROM app_positions), 0)
);

-- name: UpdateAppPosition :execrows
UPDATE app_positions
SET name = sqlc.arg(name), info = sqlc.arg(info), summary = sqlc.arg(summary), tags = sqlc.arg(tags), questions = sqlc.arg(questions), hidden = sqlc.arg(hidden),
	closed = sqlc.arg(closed), requires_main_server = sqlc.arg(requires_main_server), cooldown_secs = sqlc.arg(cooldown_secs),
	approval_note = sqlc.arg(approval_note), help_url = sqlc.narg(help_url), help_label = sqlc.narg(help_label), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: SetAppPositionOrder :exec
UPDATE app_positions SET sort_order = sqlc.arg(sort_order) WHERE id = sqlc.arg(id);

-- name: DeleteAppPosition :execrows
DELETE FROM app_positions WHERE id = $1 AND behaviour IS NULL;

-- name: CountPendingAppsForPosition :one
SELECT COUNT(*) FROM apps WHERE "position" = $1 AND state = 'pending';
