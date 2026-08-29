-- +goose Up
-- +goose StatementBegin
-- ADD COLUMN IF NOT EXISTS with a constant DEFAULT is metadata-only on
-- Postgres 11+ (no table rewrite), safe and instant regardless of table
-- size. IF NOT EXISTS added on the port -- genesis_foundational_tables.sql
-- already captures this column since prod already has it.
alter table packs add column if not exists pack_type text not null default 'bot';

-- Split into NOT VALID + a separate VALIDATE CONSTRAINT rather than a
-- single ADD CONSTRAINT ... CHECK: the single-statement form takes an
-- ACCESS EXCLUSIVE lock (blocks reads AND writes on packs) for as long as
-- it takes to scan and validate every existing row. NOT VALID applies the
-- constraint to new/future writes immediately without scanning anything;
-- VALIDATE CONSTRAINT then does the existing-row scan under a much
-- lighter SHARE UPDATE EXCLUSIVE lock, which still allows concurrent
-- reads and writes to packs while it runs.
-- Guarded on the port: genesis_foundational_tables.sql already captures
-- this constraint since prod already has it.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'packs_pack_type_check' AND conrelid = 'packs'::regclass
    ) THEN
        ALTER TABLE packs ADD CONSTRAINT packs_pack_type_check CHECK (pack_type IN ('bot', 'server', 'emoji')) NOT VALID;
        ALTER TABLE packs VALIDATE CONSTRAINT packs_pack_type_check;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/packtype.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
