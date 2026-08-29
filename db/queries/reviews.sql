-- name: GetReviews :many
SELECT id, target_type, target_id, author, owner_review, content, stars, created_at, parent_id
FROM reviews
WHERE target_id = $1 AND target_type = $2
ORDER BY created_at ASC;

-- name: GetReviewParentID :one
SELECT parent_id FROM reviews WHERE id = $1;

-- name: DeleteReviewByID :exec
DELETE FROM reviews WHERE id = $1;

-- name: CountTeamByID :one
SELECT COUNT(*) FROM teams WHERE id = $1;

-- name: CountRootReview :one
SELECT COUNT(*) FROM reviews WHERE author = $1 AND target_id = $2 AND target_type = $3 AND parent_id IS NULL;

-- name: CountReviewByID :one
SELECT COUNT(*) FROM reviews WHERE id = $1;

-- name: InsertReview :one
INSERT INTO reviews (author, target_id, target_type, content, stars, parent_id, owner_review) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: GetReviewForModify :one
SELECT author, content, stars, owner_review FROM reviews WHERE id = $1 AND target_id = $2 AND target_type = $3;

-- name: UpdateReview :exec
UPDATE reviews SET content = $1, stars = $2 WHERE id = $3;
