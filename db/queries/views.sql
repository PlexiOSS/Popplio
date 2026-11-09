-- name: RecordEntityView :exec
INSERT INTO entity_views_daily (target_type, target_id, day, views)
VALUES ($1, $2, CURRENT_DATE, 1)
ON CONFLICT (target_type, target_id, day) DO UPDATE SET views = entity_views_daily.views + 1;

-- name: GetEntityRecentViews :one
SELECT COALESCE(SUM(views), 0)::int AS views
FROM entity_views_daily
WHERE target_type = $1 AND target_id = $2 AND day > CURRENT_DATE - 30;

-- name: PruneEntityViewsDaily :exec
DELETE FROM entity_views_daily WHERE day < CURRENT_DATE - 90;
