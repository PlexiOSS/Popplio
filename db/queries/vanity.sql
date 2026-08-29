-- resolveImpl (popplio/routes/vanity/assets/resolve.go) picks one of three
-- hardcoded, never-user-controlled vanity columns to filter on
-- (target_id/code/itag) -- one explicit query per column, same reasoning as
-- db/queries/payments.sql and db/queries/reports.sql.

-- name: ResolveVanityByTargetID :one
SELECT itag, target_id, target_type, code, created_at FROM vanity WHERE target_id = $1;

-- name: ResolveVanityByCode :one
SELECT itag, target_id, target_type, code, created_at FROM vanity WHERE code = $1;

-- name: ResolveVanityByItag :one
SELECT itag, target_id, target_type, code, created_at FROM vanity WHERE itag = $1;

-- name: GetBotIDByClientID :one
SELECT bot_id FROM bots WHERE client_id = $1;

-- name: GetServerIDByServerID :one
SELECT server_id FROM servers WHERE server_id = $1;

-- name: CountVanityByCode :one
SELECT COUNT(*) FROM vanity WHERE code = $1;

-- name: CountVanityByTarget :one
SELECT COUNT(*) FROM vanity WHERE target_id = $1 AND target_type = $2;

-- name: InsertVanity :exec
INSERT INTO vanity (target_id, target_type, code) VALUES ($1, $2, $3);

-- name: UpdateVanityCode :exec
UPDATE vanity SET code = $1 WHERE target_id = $2 AND target_type = $3;
