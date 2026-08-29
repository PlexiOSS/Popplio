-- name: TableExists :one
SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = sqlc.arg('table_name')::text);

-- name: HealthCheck :one
SELECT true;
