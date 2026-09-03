-- name: GetAllTeamIDs :many
SELECT id FROM teams;

-- name: CountTeamMembers :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1;

-- name: GetTeamMembersWithFlag :many
SELECT user_id FROM team_members WHERE team_id = $1 AND flags @> ARRAY[sqlc.arg('flag')::text];

-- name: PromoteTeamMemberToOwner :exec
UPDATE team_members SET flags = $1, data_holder = $2 WHERE team_id = $3 AND user_id = $4;

-- name: CountTeamDataHoldersForTeam :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND data_holder = true;

-- name: SetTeamMemberDataHolder :exec
UPDATE team_members SET data_holder = true WHERE team_id = $1 AND user_id = $2;

-- name: GetTeamDataHolderUserID :one
SELECT user_id FROM team_members WHERE team_id = $1 AND data_holder = true LIMIT 1;

-- name: GetFirstTeamMemberUserID :one
SELECT user_id FROM team_members WHERE team_id = $1 LIMIT 1;

-- name: GetInfernoplexTeamServerMembers :many
SELECT tm.team_id, tm.user_id, s.server_id
FROM team_members tm
JOIN servers s ON s.team_owner = tm.team_id
WHERE tm.service = 'infernoplex';

-- name: DeleteInfernoplexTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2 AND service = 'infernoplex';

-- name: SetTeamVoteBanned :exec
UPDATE teams SET vote_banned = $2 WHERE id = $1;

-- name: GetTeamMemberUserIDs :many
SELECT user_id FROM team_members WHERE team_id = $1;

-- name: GetTeamMembersByTeamID :many
SELECT itag, team_id, user_id, flags, service, created_at, mentionable, data_holder
FROM team_members
WHERE team_id = $1;

-- name: GetTeamMemberFlags :one
SELECT flags FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: GetUserTeamIDs :many
SELECT team_id FROM team_members WHERE user_id = $1;

-- name: GetUserTeamMembershipsForDataTask :many
SELECT team_id, flags, data_holder FROM team_members WHERE user_id = $1;

-- name: GetTeamByID :one
SELECT id, name, short, tags, vote_banned, approximate_votes, extra_links, nsfw, vanity_ref, service, created_at, updated_at
FROM teams
WHERE id = $1;

-- name: DeleteVanityByItag :exec
DELETE FROM vanity WHERE itag = $1;

-- name: GetVanityCodeByItag :one
SELECT code FROM vanity WHERE itag = $1;

-- name: InsertVanityReturningItag :one
INSERT INTO vanity (code, target_id, target_type) VALUES ($1, $2, $3) RETURNING itag;

-- name: InsertTeamForAddServer :exec
INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'api/add_server');

-- name: InsertTeamMemberForAddServer :exec
INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'api/add_server');

-- name: InsertTeamForAddBot :exec
INSERT INTO teams (id, name, vanity_ref, service) VALUES ($1, $2, $3, 'api/add_bot');

-- name: InsertTeamMemberForAddBot :exec
INSERT INTO team_members (team_id, user_id, flags, service) VALUES ($1, $2, $3, 'api/add_bot');

-- name: InsertTeam :exec
INSERT INTO teams (id, name, short, tags, extra_links, nsfw, vanity_ref) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertTeamMemberOwner :exec
INSERT INTO team_members (team_id, user_id, flags, data_holder) VALUES ($1, $2, $3, true);

-- name: CountBotsByTeamOwner :one
SELECT COUNT(*) FROM bots WHERE team_owner = $1;

-- name: CountServersByTeamOwner :one
SELECT COUNT(*) FROM servers WHERE team_owner = $1;

-- name: SearchTeamsQueue :many
SELECT id, name, COALESCE(short, '') AS short, tags, nsfw, vote_banned
FROM teams WHERE id::text = sqlc.arg('query')::text OR name ILIKE sqlc.arg('pattern')::text ORDER BY created_at;

-- name: SearchTeamsPublic :many
SELECT id, name, COALESCE(short, '') AS short, tags, nsfw, vote_banned, approximate_votes
FROM teams
WHERE (sqlc.arg('query')::text = '' OR id::text = sqlc.arg('query')::text OR name ILIKE sqlc.arg('pattern')::text)
AND (
    cardinality(sqlc.arg('tags')::text[]) = 0
    OR (sqlc.arg('tag_mode')::text = '@>' AND tags @> sqlc.arg('tags')::text[])
    OR (sqlc.arg('tag_mode')::text = '&&' AND tags && sqlc.arg('tags')::text[])
)
AND (sqlc.arg('votes_from')::int = 0 OR approximate_votes >= sqlc.arg('votes_from')::int)
AND (sqlc.arg('votes_to')::int = 0 OR approximate_votes <= sqlc.arg('votes_to')::int)
ORDER BY approximate_votes DESC, created_at DESC
LIMIT 12;

-- name: DeleteTeamMembers :exec
DELETE FROM team_members WHERE team_id = $1;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;

-- name: InsertTeamMember :exec
INSERT INTO team_members (team_id, user_id, flags) VALUES ($1, $2, $3);

-- name: UserExistsCheck :one
SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1);

-- name: TeamMemberExists :one
SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2);

-- name: CountTeamOwnersWithFlag :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND flags && sqlc.arg('flags')::text[];

-- name: LockTeamOwnership :exec
-- Per-team advisory lock (auto-released at transaction end), acquired
-- before the "must keep at least one owner" check in delete_team_member
-- and edit_team_member. Without it, two concurrent requests stripping
-- Owner from two different members of the same two-owner team can both
-- read ownerCount=2 before either commits, both pass the guard, and the
-- team ends up with zero owners -- permanently locking out every
-- EntityOwner-gated action on it, including re-granting Owner.
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg('team_id')::text));

-- name: DeleteTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: CountTeamMembership :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND (user_id = $2 OR user_id = $3);

-- name: UpdateTeamMemberFlags :exec
UPDATE team_members SET flags = $1 WHERE team_id = $2 AND user_id = $3;

-- name: UpdateTeamMemberMentionable :exec
UPDATE team_members SET mentionable = $1 WHERE team_id = $2 AND user_id = $3;

-- name: CountTeamDataHolders :one
SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND data_holder = $2 AND user_id != $3;

-- name: UpdateTeamMemberDataHolder :exec
UPDATE team_members SET data_holder = $1 WHERE team_id = $2 AND user_id = $3;

-- name: GetTeamInfoForEdit :one
SELECT name, short, tags, extra_links, nsfw FROM teams WHERE id = $1;

-- name: TouchTeamUpdatedAt :exec
UPDATE teams SET updated_at = NOW() WHERE id = $1;

-- name: UpdateTeamName :exec
UPDATE teams SET name = $1 WHERE id = $2;

-- name: UpdateTeamShort :exec
UPDATE teams SET short = $1 WHERE id = $2;

-- name: UpdateTeamTags :exec
UPDATE teams SET tags = $1 WHERE id = $2;

-- name: UpdateTeamExtraLinks :exec
UPDATE teams SET extra_links = $1 WHERE id = $2;

-- name: UpdateTeamNsfw :exec
UPDATE teams SET nsfw = $1 WHERE id = $2;

-- name: GetTeamSEOFields :one
SELECT name, short, created_at, updated_at FROM teams WHERE id = $1;

-- name: GetTeamSEO :one
SELECT id, name, short FROM teams WHERE id = $1;
