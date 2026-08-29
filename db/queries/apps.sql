-- name: GetUserApps :many
SELECT app_id, user_id, questions, answers, state, created_at, position, review_feedback
FROM apps
WHERE user_id = $1;

-- name: GetUserAppBanned :one
SELECT app_banned FROM users WHERE user_id = $1;

-- name: CountPendingUserApps :one
SELECT COUNT(*) FROM apps WHERE user_id = $1 AND position = $2 AND state = 'pending';

-- name: GetLastAppCreatedAt :one
SELECT created_at FROM apps WHERE user_id = $1 AND position = $2 ORDER BY created_at DESC LIMIT 1;

-- name: InsertApp :exec
INSERT INTO apps (app_id, user_id, position, questions, answers) VALUES ($1, $2, $3, $4, $5);
