-- name: InsertTicket :exec
INSERT INTO tickets (id, channel_id, topic_id, issue, messages, user_id) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetTicketOwner :one
SELECT user_id FROM tickets WHERE id = $1;

-- name: GetTicket :one
SELECT id, channel_id, topic_id, issue, ticket_context, messages, user_id, close_user_id, open, created_at, enc_key
FROM tickets
WHERE id = $1;

-- name: GetUserTickets :many
SELECT id, channel_id, topic_id, issue, ticket_context, messages, user_id, close_user_id, open, created_at, enc_key
FROM tickets
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetStaffTicketsPage :many
SELECT id, channel_id, topic_id, issue, ticket_context, messages, user_id, close_user_id, open, created_at, enc_key
FROM tickets
WHERE sqlc.narg('open')::boolean IS NULL OR open = sqlc.narg('open')
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTicketOwnerAndOpen :one
SELECT user_id, open FROM tickets WHERE id = $1;

-- name: AppendTicketMessage :exec
UPDATE tickets SET messages = messages || $1 WHERE id = $2;

-- name: UpdateTicketOpenState :exec
UPDATE tickets SET open = $1, close_user_id = $2 WHERE id = $3;

-- name: CountTicketsByOpen :many
SELECT open, COUNT(*) FROM tickets GROUP BY open;
