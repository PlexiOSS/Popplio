-- name: GetPackByURL :one
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
WHERE url = $1;

-- name: GetUserPacksByOwner :many
SELECT owner, name, short, tags, url, created_at, pack_type, bots, servers, vote_banned
FROM packs
WHERE owner = $1
ORDER BY created_at DESC;

-- name: GetHomepagePacks :many
SELECT p.owner, p.name, p.short, p.tags, p.url, p.created_at, p.pack_type, p.bots, p.servers, p.vote_banned
FROM packs p
LEFT JOIN LATERAL (
    SELECT COUNT(*) FILTER (WHERE ev.upvote) - COUNT(*) FILTER (WHERE NOT ev.upvote) AS score
    FROM entity_votes ev
    WHERE ev.target_type = 'pack' AND ev.target_id = p.url AND ev.void = false
) v ON true
WHERE char_length(p.short) >= 20
ORDER BY v.score DESC, cardinality(p.bots) + cardinality(p.servers) DESC, p.created_at DESC
LIMIT 9;

-- name: BrowsePacks :many
SELECT p.owner, p.name, p.short, p.tags, p.url, p.created_at, p.pack_type, p.bots, p.servers, p.vote_banned
FROM packs p
LEFT JOIN LATERAL (
    SELECT COUNT(*) FILTER (WHERE ev.upvote) - COUNT(*) FILTER (WHERE NOT ev.upvote) AS score
    FROM entity_votes ev
    WHERE ev.target_type = 'pack' AND ev.target_id = p.url AND ev.void = false
) v ON true
WHERE (sqlc.narg('pack_type')::text IS NULL OR p.pack_type = sqlc.narg('pack_type')::text)
AND (sqlc.arg('include_all')::boolean OR char_length(p.short) >= 20)
ORDER BY
    CASE WHEN sqlc.arg('sort')::text = 'votes' THEN v.score END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'votes' THEN cardinality(p.bots) + cardinality(p.servers) END DESC,
    p.created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountBrowsePacks :one
SELECT COUNT(*)
FROM packs p
WHERE (sqlc.narg('pack_type')::text IS NULL OR p.pack_type = sqlc.narg('pack_type')::text)
AND (sqlc.arg('include_all')::boolean OR char_length(p.short) >= 20);

-- name: GetPackEmojis :many
SELECT id, name, animated, position, downloads, created_at FROM pack_emojis WHERE pack_url = $1 ORDER BY position ASC;

-- name: GetPackEmojiByID :one
-- Backs the standalone /emojis/{id} page -- globally addressable by ID
-- alone (IDs are client-generated UUIDs, not scoped per-pack), no need to
-- know the owning pack's URL up front.
SELECT id, pack_url, name, animated, position, downloads, created_at FROM pack_emojis WHERE id = $1;

-- name: IncrementPackEmojiDownloads :one
UPDATE pack_emojis SET downloads = downloads + 1 WHERE id = $1 RETURNING downloads;

-- name: GetAllPackEmojisPaged :many
-- The flat /emojis browse page -- every emoji across every pack, newest
-- first, with just enough of the owning pack to link back to it.
SELECT e.id, e.name, e.animated, e.downloads, e.created_at, e.pack_url, p.name AS pack_name
FROM pack_emojis e
JOIN packs p ON p.url = e.pack_url
ORDER BY e.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPackEmojis :one
SELECT COUNT(*) FROM pack_emojis;

-- name: GetPackSEO :one
SELECT name, short FROM packs WHERE url = $1;

-- name: CountPackByURL :one
SELECT COUNT(*) FROM packs WHERE url = $1;

-- name: InsertPack :exec
INSERT INTO packs (name, url, short, tags, bots, servers, owner, pack_type) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InsertPackEmoji :exec
-- downloads/created_at are optional (COALESCE to the column defaults when
-- not given) so a pack edit that deletes-and-reinserts an *unchanged*
-- emoji can carry its real download count and original upload date
-- forward instead of resetting both to 0/now() -- see patch_pack, which
-- passes the old row's values through for any ID that already existed.
INSERT INTO pack_emojis (id, pack_url, name, animated, position, downloads, created_at)
VALUES ($1, $2, $3, $4, $5, COALESCE(sqlc.narg('downloads')::integer, 0), COALESCE(sqlc.narg('created_at')::timestamptz, now()));

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
