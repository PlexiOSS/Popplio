-- +goose Up
-- +goose StatementBegin
-- Guarded on the port: genesis_foundational_tables.sql already captures
-- both the column and the constraint since prod already has this change.
alter table bots add column if not exists self_status text;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'bots_self_status_check' AND conrelid = 'bots'::regclass
    ) THEN
        ALTER TABLE bots ADD CONSTRAINT bots_self_status_check CHECK (self_status IS NULL OR self_status IN ('online', 'idle', 'dnd', 'offline'));
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/botselfstatus.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
