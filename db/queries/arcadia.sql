-- name: ListRPCLogs :many
SELECT id, user_id, method, data, state, created_at FROM rpc_logs ORDER BY created_at DESC;

-- name: GetRPCMethodCountsForUser :many
SELECT method, COUNT(*) FROM rpc_logs WHERE user_id = $1 GROUP BY method;

-- name: InsertRPCLog :one
INSERT INTO rpc_logs (method, user_id, data) VALUES ($1, $2, $3) RETURNING id;

-- name: CountRecentRPCLogs :one
SELECT COUNT(*) FROM rpc_logs WHERE user_id = $1 AND NOW() - created_at < INTERVAL '7 minutes';

-- name: UpdateRPCLogState :exec
UPDATE rpc_logs SET state = $1 WHERE id = $2;

-- name: DeleteAuthChainByUserID :exec
DELETE FROM staffpanel__authchain WHERE user_id = $1;

-- name: InsertStaffGeneralLog :exec
INSERT INTO staff_general_logs (user_id, action, data) VALUES ($1, $2, $3);

-- name: UpsertEntityBadge :exec
INSERT INTO entity_badges (target_type, target_id, badge_id, reason, awarded_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (target_type, target_id, badge_id) DO UPDATE SET reason = $4, awarded_by = $5, created_at = NOW();

-- name: DeleteEntityBadge :exec
DELETE FROM entity_badges WHERE target_type = $1 AND target_id = $2 AND badge_id = $3;

-- name: DeleteStaleAuthChainEntries :exec
DELETE FROM staffpanel__authchain WHERE last_seen_at < NOW() - INTERVAL '1 hour';

-- name: DeleteExpiredPendingAuthChainEntries :exec
DELETE FROM staffpanel__authchain WHERE state = 'pending' AND created_at < NOW() - INTERVAL '5 minutes';

-- name: CountAuthChainByToken :one
SELECT COUNT(*) FROM staffpanel__authchain WHERE token = $1;

-- name: GetAuthChainByToken :one
SELECT user_id, created_at, state FROM staffpanel__authchain WHERE token = $1;

-- name: GetPopplioTokenByToken :one
SELECT popplio_token FROM staffpanel__authchain WHERE token = $1;

-- name: InsertAuthChain :exec
INSERT INTO staffpanel__authchain (user_id, token, popplio_token, state) VALUES ($1, $2, $3, $4);

-- name: ActivateAuthChain :exec
UPDATE staffpanel__authchain SET state = 'active' WHERE token = $1;

-- name: DeleteAuthChainByToken :execrows
DELETE FROM staffpanel__authchain WHERE token = $1;

-- name: TouchAuthChainSeen :exec
UPDATE staffpanel__authchain SET last_seen_at = NOW() WHERE token = $1;

-- name: GetStaffPositionsAndBotFlag :one
SELECT sm.positions, COALESCE(iuc.bot, false) AS bot
FROM staff_members sm
LEFT JOIN internal_user_cache__discord iuc ON iuc.id = sm.user_id
WHERE sm.user_id = $1;

-- name: GetStaffDisciplinaries :many
SELECT d.id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
	t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
	EXTRACT(epoch FROM t.max_expiry) AS max_expiry, t.created_at AS type_created_at
FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
WHERE d.user_id = $1;

-- name: GetActiveStaffDisciplinaries :many
SELECT d.id, d.created_at, EXTRACT(epoch FROM d.expiry) AS expiry, d.title, d.description, d.type,
	t.name AS type_name, t.description AS type_description, t.self_assignable, t.perm_limits, t.additory, t.needs_approval,
	EXTRACT(epoch FROM t.max_expiry) AS max_expiry, t.created_at AS type_created_at
FROM staff_disciplinary d LEFT JOIN staff_disciplinary_types t ON t.id = d.type
WHERE d.user_id = $1 AND NOW() - d.created_at < d.expiry;

-- name: GetStaffMemberRaw :one
SELECT positions, perm_overrides, no_autosync, unaccounted, mfa_verified, created_at FROM staff_members WHERE user_id = $1;

-- name: GetStaffPositionsByIDs :many
SELECT id, name, role_id, perms, corresponding_roles, icon, index, created_at FROM staff_positions WHERE id = ANY(sqlc.arg('ids')::uuid[]);

-- name: GetTeamMembersMentionable :many
SELECT user_id, mentionable FROM team_members WHERE team_id = $1;

-- name: GetBotsOwnersByIDs :many
SELECT bot_id, owner, team_owner FROM bots WHERE bot_id = ANY(sqlc.arg('bot_ids')::text[]);

-- name: GetServersTeamOwnersByIDs :many
SELECT server_id, team_owner FROM servers WHERE server_id = ANY(sqlc.arg('server_ids')::text[]);

-- name: GetTeamMembersByTeamIDs :many
SELECT team_id, user_id, mentionable FROM team_members WHERE team_id = ANY(sqlc.arg('team_ids')::uuid[]);

-- name: ListStaffRolesOrdered :many
SELECT id::text, name, role_id, index, perms FROM staff_positions ORDER BY index;

-- name: LookupStaffRole :one
SELECT id::text, name, role_id, index, perms
FROM staff_positions
WHERE lower(name) = lower(sqlc.arg('input')) OR role_id = sqlc.arg('role_id') OR id::text = sqlc.arg('role_id')
LIMIT 1;

-- name: GetStaffRoleHolderCounts :many
SELECT sp.id::text, count(sm.user_id)
FROM staff_positions sp
LEFT JOIN staff_members sm ON sp.id = ANY(sm.positions)
GROUP BY sp.id;

-- name: UpdateStaffMemberPermOverrides :execrows
UPDATE staff_members SET perm_overrides = $1 WHERE user_id = $2;

-- name: UpdateStaffPositionPerms :execrows
UPDATE staff_positions SET perms = $1 WHERE id = $2;

-- name: GetModCasesForUser :many
SELECT action, moderator_id, reason, created_at FROM mod_cases
WHERE guild_id = $1 AND user_id = $2
ORDER BY created_at DESC LIMIT 10;

-- name: InsertModCase :exec
INSERT INTO mod_cases (guild_id, user_id, moderator_id, action, reason) VALUES ($1, $2, $3, $4, $5);

-- name: GetAllStaffPositions :many
SELECT id, name, role_id, index, perms, corresponding_roles FROM staff_positions;

-- name: GetAllStaffMembersForUpdate :many
SELECT user_id, positions, perm_overrides, no_autosync, unaccounted FROM staff_members FOR UPDATE;

-- name: RemoveStaffMemberPosition :exec
UPDATE staff_members SET positions = array_remove(positions, $1) WHERE user_id = $2;

-- name: InsertUserWithToken :exec
INSERT INTO users (user_id, api_token) VALUES ($1, $2);

-- name: UpdateStaffMemberPositions :exec
UPDATE staff_members SET positions = $1, unaccounted = false WHERE user_id = $2;

-- name: InsertStaffMemberPositions :exec
INSERT INTO staff_members (user_id, positions) VALUES ($1, $2);

-- name: DeleteStaffMember :exec
DELETE FROM staff_members WHERE user_id = $1;

-- name: MarkStaffMemberUnaccounted :exec
UPDATE staff_members SET positions = '{}', unaccounted = true WHERE user_id = $1;

-- name: GetTopReviewers :many
SELECT user_id, approved_count, denied_count, total_count FROM (
	SELECT rpc.user_id,
		SUM(CASE WHEN rpc.method = 'Approve' THEN 1 ELSE 0 END) AS approved_count,
		SUM(CASE WHEN rpc.method = 'Deny' THEN 1 ELSE 0 END) AS denied_count,
		SUM(CASE WHEN rpc.method IN ('Approve', 'Deny') THEN 1 ELSE 0 END) AS total_count
	FROM rpc_logs rpc
	LEFT JOIN staff_members sm ON rpc.user_id = sm.user_id
	WHERE rpc.method IN ('Approve', 'Deny') AND sm.user_id IS NOT NULL
	GROUP BY rpc.user_id
) AS subquery
WHERE total_count > 0
ORDER BY total_count DESC
LIMIT $1;

-- name: GetCachedDiscordUsername :one
SELECT username FROM internal_user_cache__discord WHERE id = $1;

-- name: CountStaffPositionByNameOrRoleID :one
SELECT EXISTS(SELECT 1 FROM staff_positions WHERE name = $1 OR role_id = $2);

-- name: GetNextStaffPositionIndex :one
SELECT COALESCE(MAX(index), 0) + 1 FROM staff_positions;

-- name: InsertStaffPosition :exec
INSERT INTO staff_positions (name, role_id, perms, index) VALUES ($1, $2, '{}', $3);

-- name: RenameStaffPosition :exec
UPDATE staff_positions SET name = $1 WHERE id = $2;

-- name: DeleteStaffPosition :exec
DELETE FROM staff_positions WHERE id = $1;

-- name: CountBotsByType :many
SELECT type AS method, COUNT(*) FROM bots GROUP BY type;

-- name: GetAnalyticsCounts :one
SELECT
	(SELECT COUNT(*) FROM bots) AS bots,
	(SELECT COUNT(*) FROM teams) AS teams,
	(SELECT COUNT(*) FROM users) AS users,
	(SELECT COUNT(*) FROM servers) AS servers,
	(SELECT COUNT(*) FROM packs) AS packs;

-- name: GetPendingBotsQueue :many
SELECT claimed_by, bot_id, approval_note, short, invite, client_id
FROM bots WHERE type = 'pending' ORDER BY created_at ASC;

-- name: GetUserOwnedEntities :many
SELECT b.bot_id as id, b.type, 'bot' as entity
FROM bots b
WHERE b.team_owner IN (SELECT tm.team_id FROM team_members tm WHERE tm.user_id = sqlc.arg('user_id')::text)
   OR b.owner = sqlc.arg('user_id')::text

UNION

SELECT s.server_id as id, s.type, 'server' as entity
FROM servers s
WHERE s.team_owner IN (SELECT tm.team_id FROM team_members tm WHERE tm.user_id = sqlc.arg('user_id')::text)

UNION

SELECT p.url as id, 'pack' as type, 'pack' as entity
FROM packs p
WHERE p.owner = sqlc.arg('user_id')::text;
