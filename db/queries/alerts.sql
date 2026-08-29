-- name: InsertAlert :exec
INSERT INTO alerts (user_id, type, url, message, title, icon, alert_data, priority, category)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetUserAlertByItag :one
SELECT itag, url, message, type, title, created_at, acked, alert_data, icon, priority, category
FROM alerts
WHERE user_id = $1 AND itag = $2;

-- name: GetUserAlertsPage :many
SELECT itag, url, message, type, title, created_at, acked, alert_data, icon, priority, category
FROM alerts
WHERE user_id = $1
ORDER BY created_at DESC, priority ASC
LIMIT $2 OFFSET $3;

-- name: GetUserAlertsByAcked :many
SELECT itag, url, message, type, title, created_at, acked, alert_data, icon, priority, category
FROM alerts
WHERE user_id = $1 AND acked = $2
ORDER BY created_at DESC, priority ASC
LIMIT $3;

-- name: CountUserAlerts :one
SELECT COUNT(*) FROM alerts WHERE user_id = $1;

-- name: CountUnackedUserAlerts :one
SELECT COUNT(*) FROM alerts WHERE user_id = $1 AND acked = false;

-- name: AckAlert :exec
UPDATE alerts SET acked = true WHERE user_id = $1 AND itag = $2;

-- name: UnackAlert :exec
UPDATE alerts SET acked = false WHERE user_id = $1 AND itag = $2;

-- name: DeleteUserAlert :exec
DELETE FROM alerts WHERE user_id = $1 AND itag = $2;

-- name: DeleteAllUserAlerts :exec
DELETE FROM alerts WHERE user_id = $1;
