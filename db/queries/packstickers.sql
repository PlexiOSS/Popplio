-- Sticker packs' counterpart to the pack_emojis queries in packs.sql --
-- kept in their own file rather than folded into packs.sql, since this is
-- a whole second set of CRUD + standalone-lookup queries, not one or two
-- additions.

-- name: GetPackStickers :many
SELECT id, name, animated, position, downloads, created_at FROM pack_stickers WHERE pack_url = $1 ORDER BY position ASC;

-- name: InsertPackSticker :exec
-- downloads/created_at are optional -- see InsertPackEmoji's own comment
-- in packs.sql, same reasoning.
INSERT INTO pack_stickers (id, pack_url, name, animated, position, downloads, created_at)
VALUES ($1, $2, $3, $4, $5, COALESCE(sqlc.narg('downloads')::integer, 0), COALESCE(sqlc.narg('created_at')::timestamptz, now()));

-- name: DeletePackStickers :exec
DELETE FROM pack_stickers WHERE pack_url = $1;

-- name: GetPackStickerByID :one
-- Backs the standalone /stickers/{id} page -- see GetPackEmojiByID's own
-- comment, same reasoning.
SELECT id, pack_url, name, animated, position, downloads, created_at FROM pack_stickers WHERE id = $1;

-- name: IncrementPackStickerDownloads :one
UPDATE pack_stickers SET downloads = downloads + 1 WHERE id = $1 RETURNING downloads;

-- name: GetAllPackStickersPaged :many
SELECT s.id, s.name, s.animated, s.downloads, s.created_at, s.pack_url, p.name AS pack_name
FROM pack_stickers s
JOIN packs p ON p.url = s.pack_url
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPackStickers :one
SELECT COUNT(*) FROM pack_stickers;
