-- name: GetUserNotifications :many
SELECT endpoint, notif_id, created_at, ua
FROM user_notifications
WHERE user_id = $1;

-- name: GetUserNotificationSubscriptions :many
SELECT notif_id, auth, endpoint, p256dh
FROM user_notifications
WHERE user_id = $1;

-- name: DeleteUserNotificationByNotifID :exec
DELETE FROM user_notifications WHERE notif_id = $1;

-- name: GetUserNotificationPrefEnabled :one
SELECT enabled FROM user_notification_prefs WHERE user_id = $1 AND category = $2;

-- name: DeleteUserNotificationByEndpoint :exec
DELETE FROM user_notifications WHERE user_id = $1 AND endpoint = $2;

-- name: InsertUserNotification :exec
INSERT INTO user_notifications (user_id, notif_id, auth, p256dh, endpoint, ua) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountUserNotification :one
SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND notif_id = $2;

-- name: DeleteUserNotification :exec
DELETE FROM user_notifications WHERE user_id = $1 AND notif_id = $2;

-- name: GetUserNotificationPrefs :many
SELECT category, enabled FROM user_notification_prefs WHERE user_id = $1;

-- name: UpsertUserNotificationPref :exec
INSERT INTO user_notification_prefs (user_id, category, enabled) VALUES ($1, $2, $3) ON CONFLICT (user_id, category) DO UPDATE SET enabled = $3;
