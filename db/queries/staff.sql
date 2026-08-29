-- name: CountAuthChainToken :one
SELECT COUNT(*) FROM staffpanel__authchain WHERE popplio_token = $1 AND user_id = $2;

-- name: GetAppsList :many
SELECT app_id, user_id, questions, answers, state, created_at, position, review_feedback
FROM apps
WHERE (sqlc.narg('user_id')::text IS NULL OR user_id = sqlc.narg('user_id'))
ORDER BY created_at DESC;

-- name: GetAppByID :one
SELECT app_id, user_id, questions, answers, state, created_at, position, review_feedback
FROM apps
WHERE app_id = $1;

-- name: DeleteApp :exec
DELETE FROM apps WHERE app_id = $1;

-- name: ApproveApp :exec
UPDATE apps SET state = 'approved', review_feedback = $2 WHERE app_id = $1;

-- name: DenyApp :exec
UPDATE apps SET state = 'denied', review_feedback = $2 WHERE app_id = $1;

-- name: CountStaffMemberByID :one
SELECT COUNT(*) FROM staff_members WHERE user_id = $1;

-- name: GetStaffPositionCount :one
SELECT cardinality(positions) FROM staff_members WHERE user_id = $1;

-- name: GetStaffMembersRoster :many
SELECT user_id, positions FROM staff_members;

-- name: GetStaffPositionsRoster :many
SELECT id, name, icon, index FROM staff_positions;

-- name: GetRecentShopPurchases :many
SELECT id, target_type, target_id, item_id, cents, created_at
FROM shop_purchases
ORDER BY created_at DESC
LIMIT $1;

-- name: GetStaffMFA :one
SELECT mfa_secret, mfa_verified FROM staff_members WHERE user_id = $1;

-- name: UpdateStaffMFASecret :exec
UPDATE staff_members SET mfa_secret = $1 WHERE user_id = $2;

-- name: UpdateStaffMFAClear :exec
UPDATE staff_members SET mfa_secret = NULL, mfa_verified = FALSE WHERE user_id = $1;

-- name: UpdateStaffMFAVerified :exec
UPDATE staff_members SET mfa_verified = TRUE WHERE user_id = $1;

-- name: ListStaffMemberIDs :many
SELECT user_id FROM staff_members;

-- name: GetStaffMemberForEdit :one
SELECT perm_overrides, no_autosync, unaccounted FROM staff_members WHERE user_id = $1 FOR UPDATE;

-- name: UpdateStaffMemberEdit :exec
UPDATE staff_members SET perm_overrides = $1, no_autosync = $2, unaccounted = $3 WHERE user_id = $4;

-- Staff templates: pre-built answers staff pick from when approving/denying.

-- name: ListStaffTemplates :many
SELECT id, name, emoji, tags, description, type, entity_type, created_at
FROM staff_templates ORDER BY created_at DESC;

-- name: InsertStaffTemplate :exec
INSERT INTO staff_templates (name, emoji, tags, description, type, entity_type) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountStaffTemplateByID :one
SELECT EXISTS(SELECT 1 FROM staff_templates WHERE id::text = $1);

-- name: UpdateStaffTemplate :exec
UPDATE staff_templates SET name = $1, emoji = $2, tags = $3, description = $4, type = $5, entity_type = $6 WHERE id::text = $7;

-- name: DeleteStaffTemplate :exec
DELETE FROM staff_templates WHERE id::text = $1;

-- Disciplinary types: the punishments that can be issued.

-- name: ListStaffDisciplinaryTypes :many
SELECT id, name, description, self_assignable, perm_limits, additory, needs_approval, EXTRACT(epoch FROM max_expiry) AS max_expiry, created_at
FROM staff_disciplinary_types ORDER BY created_at DESC;

-- name: InsertStaffDisciplinaryType :exec
INSERT INTO staff_disciplinary_types (id, name, description, self_assignable, perm_limits, additory, needs_approval, max_expiry)
VALUES ($1, $2, $3, $4, $5, $6, $7, make_interval(secs => sqlc.narg('secs')::float8));

-- name: CountStaffDisciplinaryTypeByID :one
SELECT EXISTS(SELECT 1 FROM staff_disciplinary_types WHERE id = $1);

-- name: UpdateStaffDisciplinaryType :exec
UPDATE staff_disciplinary_types SET name = $1, description = $2, self_assignable = $3, perm_limits = $4, additory = $5, needs_approval = $6, max_expiry = make_interval(secs => sqlc.narg('secs')::float8) WHERE id = $7;

-- name: DeleteStaffDisciplinaryType :exec
DELETE FROM staff_disciplinary_types WHERE id = $1;

-- Staff positions: the panel's own full-column listing and index shuffling.
-- (See db/queries/arcadia.sql for the bot-facing staff_positions queries.)

-- name: ListStaffPositionsFull :many
SELECT id, name, role_id, perms, corresponding_roles, icon, index, created_at
FROM staff_positions ORDER BY index ASC;

-- name: GetStaffPositionIndexByIDText :one
SELECT index FROM staff_positions WHERE id::text = $1;

-- name: UpdateStaffPositionIndexByIDText :exec
UPDATE staff_positions SET index = $1 WHERE id::text = $2;

-- name: GetStaffPositionIndexByID :one
SELECT index FROM staff_positions WHERE id = $1;

-- name: GetStaffPositionForUpdate :one
SELECT perms, index, role_id FROM staff_positions WHERE id = $1 FOR UPDATE;

-- name: ShiftStaffPositionIndexesFrom :exec
UPDATE staff_positions SET index = index + 1 WHERE index >= $1;

-- name: ShiftStaffPositionIndexesAfter :exec
UPDATE staff_positions SET index = index - 1 WHERE index > $1;

-- name: InsertStaffPositionFull :exec
INSERT INTO staff_positions (name, perms, corresponding_roles, icon, role_id, index) VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateStaffPositionFull :exec
UPDATE staff_positions SET name = $1, perms = $2, corresponding_roles = $3, role_id = $4, icon = $5 WHERE id = $6;
