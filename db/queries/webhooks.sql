-- name: GetWebhooks :many
SELECT id, name, target_id, target_type, url, broken, failed_requests, simple_auth, hmac_auth, event_whitelist, created_at
FROM webhooks
WHERE target_id = $1 AND target_type = $2;

-- name: CountWebhooksForTarget :one
SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2;

-- name: InsertWebhook :exec
INSERT INTO webhooks (target_id, target_type, url, secret, simple_auth, hmac_auth, name, event_whitelist) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: CountWebhookByID :one
SELECT COUNT(*) FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE target_id = $1 AND target_type = $2 AND id = $3;

-- name: UpdateWebhook :exec
UPDATE webhooks SET name = $1, url = $2, secret = $3, event_whitelist = $4, simple_auth = $5, hmac_auth = $6, broken = false, failed_requests = 0 WHERE target_id = $7 AND target_type = $8 AND id = $9;

-- name: GetPendingWebhookLogsForPull :many
SELECT id::text AS id, target_id, user_id, data
FROM webhook_logs
WHERE state = $1 AND target_type = $2 AND bad_intent = false;

-- name: MarkWebhookLogNoWebhooks :exec
UPDATE webhook_logs SET state = $1 WHERE id = $2;

-- name: IncrementWebhookLogTries :exec
UPDATE webhook_logs SET state = $1, tries = tries + 1 WHERE id = $2;

-- name: GetWebhooksForTarget :many
SELECT id::text AS id, secret, url, broken, failed_requests, simple_auth, hmac_auth, event_whitelist
FROM webhooks
WHERE target_id = $1 AND target_type = $2;

-- name: InsertWebhookLogReturningID :one
INSERT INTO webhook_logs (target_id, target_type, user_id, url, data, bad_intent, webhook_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id::text;

-- name: UpdateWebhookLogResponse :exec
UPDATE webhook_logs SET response = $1, status_code = $2, request_headers = $3, response_headers = $4
WHERE id = $5;

-- name: MarkWebhookBrokenForTarget :exec
UPDATE webhooks SET broken = true WHERE target_id = $1 AND target_type = $2;

-- name: IncrementWebhookFailedRequests :exec
UPDATE webhooks SET failed_requests = failed_requests + 1 WHERE target_id = $1 AND target_type = $2;

-- name: GetWebhookLogsPage :many
SELECT id, webhook_id, target_id, target_type, user_id, url, data, response, created_at, state, tries, last_try, bad_intent, status_code, request_headers, response_headers
FROM webhook_logs
WHERE target_id = $1 AND target_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountWebhookLogs :one
SELECT COUNT(*) FROM webhook_logs WHERE target_id = $1 AND target_type = $2;
