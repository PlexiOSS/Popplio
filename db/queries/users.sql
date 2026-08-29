-- name: GetUserLastBoosterClaim :one
SELECT last_booster_claim FROM users WHERE user_id = $1;

-- name: UpdateUserLastBoosterClaim :exec
UPDATE users SET last_booster_claim = NOW() WHERE user_id = $1;

-- name: GetUserAboutAndTimestamps :one
SELECT about, created_at, updated_at FROM users WHERE user_id = $1;

-- name: GetBannedUserIDs :many
SELECT user_id FROM users WHERE banned = true;

-- name: InsertBannedUser :exec
INSERT INTO users (user_id, banned, api_token) VALUES ($1, $2, $3);

-- name: ClearAllBugHunters :exec
UPDATE users SET bug_hunters = false;

-- name: SetUserBugHunter :exec
UPDATE users SET bug_hunters = true WHERE user_id = $1;

-- name: UpdateUserAppBanned :exec
UPDATE users SET app_banned = $2 WHERE user_id = $1;

-- name: UpdateUserBanned :execrows
UPDATE users SET banned = $2 WHERE user_id = $1;

-- name: GetUserByID :one
SELECT itag, user_id, experiments, certified, developer, bug_hunters, captcha_sponsor_enabled, extra_links, about, vote_banned, banned, created_at, updated_at, last_booster_claim
FROM users
WHERE user_id = $1;

-- name: GetUserAboutAndID :one
SELECT about, user_id FROM users WHERE user_id = $1;

-- name: GetUserPerm :one
SELECT experiments, banned, captcha_sponsor_enabled, vote_banned FROM users WHERE user_id = $1;

-- name: UpdateUsersUpdatedAt :exec
UPDATE users SET updated_at = NOW() WHERE user_id = $1;

-- name: UpdateUserExtraLinks :exec
UPDATE users SET extra_links = $1 WHERE user_id = $2;

-- name: UpdateUserAbout :exec
UPDATE users SET about = $1 WHERE user_id = $2;

-- name: UpdateUserCaptchaSponsorEnabled :exec
UPDATE users SET captcha_sponsor_enabled = $1 WHERE user_id = $2;

-- name: DeleteUserByID :exec
DELETE FROM users WHERE user_id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: SearchUsersQueue :many
SELECT users.user_id, users.banned, users.vote_banned
FROM users
INNER JOIN internal_user_cache__discord discord_users ON users.user_id = discord_users.id
WHERE users.user_id = sqlc.arg('query')::text OR discord_users.username ILIKE sqlc.arg('pattern')::text
ORDER BY users.created_at;
