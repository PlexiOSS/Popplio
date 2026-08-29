-- name: GetUserReminders :many
SELECT user_id, target_type, target_id, created_at, last_acked
FROM user_reminders
WHERE user_id = $1;

-- name: GetStaleUserReminders :many
SELECT user_id, target_id, target_type
FROM user_reminders
WHERE NOW() - last_acked > interval '4 hours';

-- name: TouchUserReminder :exec
UPDATE user_reminders SET last_acked = NOW() WHERE user_id = $1 AND target_id = $2 AND target_type = $3;

-- name: CountUserReminder :one
SELECT COUNT(*) FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3;

-- name: InsertUserReminder :exec
INSERT INTO user_reminders (user_id, target_id, target_type) VALUES ($1, $2, $3);

-- name: DeleteUserReminder :exec
DELETE FROM user_reminders WHERE user_id = $1 AND target_id = $2 AND target_type = $3;
