-- +goose Up
-- +goose StatementBegin
-- None of this is captured in any exp/*.sql file -- discovered by running
-- `migrate validate` against a freshly-built test database and finding it
-- disagreed with prod's live schema. exp/votesv2.sql (ported as
-- 20260801000200_votesv2.sql) originally created this table with a
-- primary key column named `id`; prod's actual entity_votes table has
-- neither that column name nor these four columns, all added ad hoc at
-- some point with no tracked record of when or why:
--   itag PRIMARY KEY DEFAULT uuid_generate_v4()  (renamed from `id`)
--   vote_num       INTEGER NOT NULL DEFAULT 0
--   voided_at      TIMESTAMPTZ
--   immutable      BOOLEAN NOT NULL DEFAULT FALSE
--   credit_redeem  UUID REFERENCES entity_vote_redeem_logs(id) ON UPDATE SET NULL ON DELETE SET NULL
-- Guarded so this is a no-op on a database that already has these
-- (i.e. prod itself, if this were ever run there -- it won't be; like
-- every other historical port, this gets bookkeeping-marked-applied on
-- prod rather than executed).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'entity_votes' AND column_name = 'id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'entity_votes' AND column_name = 'itag'
    ) THEN
        ALTER TABLE entity_votes RENAME COLUMN id TO itag;
    END IF;
END
$$;

ALTER TABLE entity_votes ADD COLUMN IF NOT EXISTS vote_num INTEGER NOT NULL DEFAULT 0;
ALTER TABLE entity_votes ADD COLUMN IF NOT EXISTS voided_at TIMESTAMPTZ;
ALTER TABLE entity_votes ADD COLUMN IF NOT EXISTS immutable BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE entity_votes ADD COLUMN IF NOT EXISTS credit_redeem UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'entity_votes_credit_fkey' AND conrelid = 'entity_votes'::regclass
    ) THEN
        ALTER TABLE entity_votes
            ADD CONSTRAINT entity_votes_credit_fkey
            FOREIGN KEY (credit_redeem) REFERENCES entity_vote_redeem_logs(id)
            ON UPDATE SET NULL ON DELETE SET NULL;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- No rollback written -- this reflects an already-long-applied, previously
-- untracked prod change, not new work.
SELECT 1;
