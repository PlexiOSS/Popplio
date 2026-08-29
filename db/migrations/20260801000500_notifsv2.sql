-- +goose Up
-- +goose StatementBegin
-- Guarded: on a fresh database, genesis_foundational_tables.sql already
-- creates user_notifications directly (it captured prod's live schema,
-- which already reflects this rename), so poppypaw never exists there --
-- this block becomes a no-op. On prod itself this file is only
-- bookkeeping-marked-applied, never re-run, so this guard only matters
-- for a from-scratch dry run.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'poppypaw') THEN
        ALTER TABLE poppypaw RENAME TO user_notifications;
        CREATE VIEW poppypaw AS SELECT * FROM user_notifications;
    END IF;
END
$$;

-- TODO, after migration remove poppypaw view
-- (Already done on prod by the time genesis_foundational_tables.sql was
-- captured -- no poppypaw view exists in that dump.)
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/notifsv2.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
