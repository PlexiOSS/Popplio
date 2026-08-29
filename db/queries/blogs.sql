-- name: GetBlogList :many
SELECT slug, title, description, user_id, created_at, draft, tags
FROM blogs
WHERE draft = false
ORDER BY created_at DESC;

-- name: GetBlogPost :one
SELECT slug, title, description, user_id, created_at, content, draft, tags
FROM blogs
WHERE slug = $1;

-- name: GetBlogSEO :one
SELECT title, description FROM blogs WHERE slug = $1;

-- Panel blog CRUD (arcadia/panel ops_content.go) sees drafts too, which is
-- why it can't reuse GetBlogList above.

-- name: ListBlogEntriesFull :many
SELECT itag, slug, title, description, user_id, content, created_at, draft, tags
FROM blogs ORDER BY created_at DESC;

-- name: InsertBlogEntry :exec
INSERT INTO blogs (slug, title, description, content, tags, user_id) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CountBlogByItag :one
SELECT EXISTS(SELECT 1 FROM blogs WHERE itag = $1);

-- name: GetBlogDraftByItag :one
SELECT draft FROM blogs WHERE itag = $1;

-- name: UpdateBlogEntry :exec
UPDATE blogs SET slug = $2, title = $3, description = $4, content = $5, tags = $6, draft = $7 WHERE itag = $1;

-- name: DeleteBlogByItag :exec
DELETE FROM blogs WHERE itag = $1;
