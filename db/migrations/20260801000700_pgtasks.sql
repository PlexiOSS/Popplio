-- +goose Up
-- +goose StatementBegin
CREATE TABLE tasks (
    task_id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
    task_key TEXT,
    task_name TEXT NOT NULL,
    output JSONB,
    statuses JSONB[] NOT NULL DEFAULT '{}',
    for_user TEXT,
    allow_unauthenticated BOOLEAN NOT NULL DEFAULT FALSE, -- If set, api tokens are not needed to access this task
    expiry INTERVAL NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/pgtasks.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
