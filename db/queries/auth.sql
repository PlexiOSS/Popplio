-- name: InsertSession :one
INSERT INTO api_sessions (token, target_id, target_type, name, type, expiry, perm_limits) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: DeleteExpiredSessions :exec
DELETE FROM api_sessions WHERE expiry < NOW();

-- name: GetSessionByToken :one
SELECT id, target_id, target_type, perm_limits FROM api_sessions WHERE token = $1;

-- name: GetFullSessionByToken :one
SELECT id, name, created_at, type, target_type, target_id, perm_limits, expiry FROM api_sessions WHERE token = $1;

-- name: GetSessions :many
SELECT id, name, created_at, type, target_type, target_id, perm_limits, expiry
FROM api_sessions
WHERE target_id = $1 AND target_type = $2;

-- name: CountSession :one
SELECT COUNT(*) FROM api_sessions WHERE target_type = $1 AND target_id = $2 AND id = $3;

-- name: DeleteSession :exec
DELETE FROM api_sessions WHERE id = $1 AND target_id = $2 AND target_type = $3;

-- name: GetUserBanStatus :one
SELECT banned, bug_hunters FROM users WHERE user_id = $1;

-- name: GetUserBanFlags :one
SELECT banned, vote_banned, app_banned FROM users WHERE user_id = $1;

-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1);

-- api_token has no default on this table (confirmed against prod: NOT NULL,
-- no default, no trigger) -- it must be supplied by the caller. The
-- original raw-SQL version of this insert never supplied it, which would
-- have failed for any genuinely brand-new user; caught by smoke-testing
-- this against the test database. See cmd/kitehelper/validatetable/validate.go's
-- createUser for the existing crypto.RandString(256) convention this now matches.
-- name: InsertUser :exec
INSERT INTO users (user_id, api_token, extra_links, developer, certified) VALUES ($1, $2, $3, false, false);

-- name: GetUserBanned :one
SELECT banned FROM users WHERE user_id = $1;

-- name: GetUserBugHunter :one
SELECT bug_hunters FROM users WHERE user_id = $1;

-- name: InsertLoginSession :one
INSERT INTO api_sessions (target_type, target_id, type, token, expiry, name) VALUES ('user', $1, 'login', $2, NOW() + INTERVAL '30 days', $3) RETURNING id;
