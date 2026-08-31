-- name: GetPackByURL :one
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
WHERE url = $1;

-- name: GetUserPacksByOwner :many
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
WHERE owner = $1
ORDER BY created_at DESC;

-- name: GetRecentPacks :many
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
ORDER BY created_at DESC
LIMIT 12;

-- name: GetAllPacks :many
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllPacksByType :many
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
WHERE pack_type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPacks :one
SELECT COUNT(*) FROM packs;

-- name: CountPacksByType :one
SELECT COUNT(*) FROM packs WHERE pack_type = $1;

-- name: GetPackEmojis :many
SELECT id, name, animated, position FROM pack_emojis WHERE pack_url = $1 ORDER BY position ASC;

-- name: GetPackSEO :one
SELECT name, short FROM packs WHERE url = $1;

-- name: CountPackByURL :one
SELECT COUNT(*) FROM packs WHERE url = $1;

-- name: InsertPack :exec
INSERT INTO packs (name, url, short, tags, bots, servers, owner, pack_type) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InsertPackEmoji :exec
INSERT INTO pack_emojis (id, pack_url, name, animated, position) VALUES ($1, $2, $3, $4, $5);

-- name: SetPackVoteBanned :exec
UPDATE packs SET vote_banned = $2 WHERE url = $1;

-- name: GetPackOwner :one
SELECT owner FROM packs WHERE url = $1;

-- name: DeletePack :exec
DELETE FROM packs WHERE url = $1;

-- name: GetPackOwnerAndType :one
SELECT owner, pack_type FROM packs WHERE url = $1;

-- name: UpdatePack :exec
UPDATE packs SET name = $1, short = $2, tags = $3, bots = $4, servers = $5 WHERE url = $6;

-- name: SearchPacksQueue :many
SELECT url, name, short, pack_type, owner, tags, vote_banned
FROM packs WHERE url = sqlc.arg('query')::text OR name ILIKE sqlc.arg('pattern')::text ORDER BY created_at;

-- name: SearchPacksPublic :many
SELECT url, name, short, pack_type, owner, tags, bots, servers, vote_banned
FROM packs
WHERE (sqlc.arg('query')::text = '' OR url = sqlc.arg('query')::text OR name ILIKE sqlc.arg('pattern')::text)
AND (
    cardinality(sqlc.arg('tags')::text[]) = 0
    OR (sqlc.arg('tag_mode')::text = '@>' AND tags @> sqlc.arg('tags')::text[])
    OR (sqlc.arg('tag_mode')::text = '&&' AND tags && sqlc.arg('tags')::text[])
)
ORDER BY created_at DESC
LIMIT 12;

-- name: DeletePackEmojis :exec
DELETE FROM pack_emojis WHERE pack_url = $1;
