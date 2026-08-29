-- name: DeleteExpiredTasks :exec
DELETE FROM tasks WHERE created_at + expiry < NOW();

-- name: GetTask :one
SELECT task_id, task_key, allow_unauthenticated, task_name, output, statuses, for_user, expiry, state, created_at
FROM tasks
WHERE task_id = $1;

-- name: InsertTask :one
INSERT INTO tasks (task_name, task_key, for_user, expiry, output, allow_unauthenticated)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING task_id;

-- name: DeleteOldDataTasks :exec
DELETE FROM tasks WHERE task_name = $1 AND task_id != $2 AND for_user = $3;

-- name: UpdateTaskState :exec
UPDATE tasks SET state = $1 WHERE task_id = $2;

-- name: UpdateTaskOutputAndState :exec
UPDATE tasks SET output = $1, state = $2 WHERE task_id = $3;

-- name: AppendTaskStatus :exec
UPDATE tasks SET statuses = array_append(statuses, $1) WHERE task_id = $2;
