-- name: ListAllStaffDisciplinaries :many
SELECT d.id, d.user_id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
	d.state, d.issued_by, d.starts_at, d.decided_by, d.decided_at, d.decision_note, d.revoked_by, d.revoked_at, d.revoke_reason,
	t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
	EXTRACT(epoch FROM t.max_expiry) AS max_expiry, t.created_at AS type_created_at
FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
WHERE (sqlc.narg(state)::text IS NULL OR d.state = sqlc.narg(state)::text)
ORDER BY d.created_at DESC
LIMIT 500;

-- name: GetStaffDisciplinaryByID :one
SELECT d.id, d.user_id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
	d.state, d.issued_by, d.starts_at, d.decided_by, d.decided_at, d.decision_note, d.revoked_by, d.revoked_at, d.revoke_reason,
	t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
	EXTRACT(epoch FROM t.max_expiry) AS max_expiry, t.created_at AS type_created_at
FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
WHERE d.id = $1;

-- name: InsertStaffDisciplinary :one
INSERT INTO staff_disciplinary (user_id, title, description, type, state, expiry, issued_by, starts_at)
VALUES (
	sqlc.arg(user_id), sqlc.arg(title), sqlc.arg(description), sqlc.arg(type), sqlc.arg(state),
	CASE WHEN sqlc.narg(expiry_secs)::bigint IS NULL THEN NULL ELSE make_interval(secs => sqlc.narg(expiry_secs)::bigint) END,
	sqlc.arg(issued_by), sqlc.narg(starts_at)
)
RETURNING id;

-- name: DecideStaffDisciplinary :execrows
UPDATE staff_disciplinary
SET state = sqlc.arg(state), decided_by = sqlc.arg(decided_by), decided_at = NOW(), decision_note = sqlc.narg(decision_note),
	starts_at = CASE WHEN sqlc.arg(state) = 'active' THEN NOW() ELSE starts_at END
WHERE id = sqlc.arg(id) AND state = 'pending';

-- name: RevokeStaffDisciplinary :execrows
UPDATE staff_disciplinary
SET state = 'revoked', revoked_by = sqlc.arg(revoked_by), revoked_at = NOW(), revoke_reason = sqlc.narg(revoke_reason)
WHERE id = sqlc.arg(id) AND state IN ('pending', 'active');
