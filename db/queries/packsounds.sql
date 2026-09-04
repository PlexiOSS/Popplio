-- Sound packs' counterpart to the pack_stickers/pack_emojis queries in
-- packs.sql/packstickers.sql -- same shape, with `duration_ms` standing in
-- for `animated` since it's the audio-specific field that matters here.

-- name: GetPackSounds :many
SELECT id, name, duration_ms, position, downloads, created_at FROM pack_sounds WHERE pack_url = $1 ORDER BY position ASC;

-- name: InsertPackSound :exec
-- downloads/created_at are optional -- see InsertPackEmoji's own comment
-- in packs.sql, same reasoning.
INSERT INTO pack_sounds (id, pack_url, name, duration_ms, position, downloads, created_at)
VALUES ($1, $2, $3, $4, $5, COALESCE(sqlc.narg('downloads')::integer, 0), COALESCE(sqlc.narg('created_at')::timestamptz, now()));

-- name: DeletePackSounds :exec
DELETE FROM pack_sounds WHERE pack_url = $1;

-- name: GetPackSoundByID :one
-- Backs the standalone /sounds/{id} page -- see GetPackEmojiByID's own
-- comment, same reasoning.
SELECT id, pack_url, name, duration_ms, position, downloads, created_at FROM pack_sounds WHERE id = $1;

-- name: IncrementPackSoundDownloads :one
UPDATE pack_sounds SET downloads = downloads + 1 WHERE id = $1 RETURNING downloads;

-- name: GetAllPackSoundsPaged :many
SELECT s.id, s.name, s.duration_ms, s.downloads, s.created_at, s.pack_url, p.name AS pack_name
FROM pack_sounds s
JOIN packs p ON p.url = s.pack_url
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPackSounds :one
SELECT COUNT(*) FROM pack_sounds;
