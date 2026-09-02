-- name: GetServerTemplateReactionCounts :one
SELECT
    COUNT(*) FILTER (WHERE liked) AS likes,
    COUNT(*) FILTER (WHERE NOT liked) AS dislikes
FROM server_template_reactions
WHERE template_id = $1;

-- name: GetUserServerTemplateReaction :one
SELECT liked FROM server_template_reactions WHERE template_id = $1 AND user_id = $2;

-- name: UpsertServerTemplateReaction :exec
INSERT INTO server_template_reactions (template_id, user_id, liked)
VALUES ($1, $2, $3)
ON CONFLICT (template_id, user_id) DO UPDATE SET liked = $3;

-- name: DeleteServerTemplateReaction :exec
DELETE FROM server_template_reactions WHERE template_id = $1 AND user_id = $2;
