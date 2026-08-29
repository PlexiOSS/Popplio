-- +goose Up
-- +goose StatementBegin
-- Not captured in any exp/*.sql file -- discovered the same way as
-- 20260801000250_entity_votes_undocumented_columns.sql: `migrate validate`
-- against a freshly-built test database disagreed with prod's live
-- schema. prod's partners table has a nullable bot_id TEXT column with a
-- FK to bots(bot_id), added ad hoc with no tracked record of when or why.
ALTER TABLE partners ADD COLUMN IF NOT EXISTS bot_id TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'partners_bot_id_fkey' AND conrelid = 'partners'::regclass
    ) THEN
        ALTER TABLE partners
            ADD CONSTRAINT partners_bot_id_fkey
            FOREIGN KEY (bot_id) REFERENCES bots(bot_id)
            ON UPDATE CASCADE ON DELETE CASCADE;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- No rollback written -- this reflects an already-long-applied, previously
-- untracked prod change, not new work.
SELECT 1;
