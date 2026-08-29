-- +goose Up
-- +goose StatementBegin
-- itag PRIMARY KEY added on the port: the original exp/vanityv2.sql left
-- itag with no PK/unique constraint at all, yet prod's live table has
-- "vanity_pkey" PRIMARY KEY (itag) -- confirmed via \d vanity against
-- prod. It must have been added by hand at some point outside any
-- tracked script. Without it, bots/servers/teams' vanity_ref FK (added in
-- 20260801004500_add_deferred_foreign_keys.sql) can't be created at all
-- ("no unique constraint matching given keys for referenced table").
CREATE TABLE vanity (
    itag UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    code CITEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (target_id, target_type)
);

-- After migration, add a foreign key to bot that references the vanity to ensure that all bots have a vanity
-- This should be called vanity_ref
-- (Note: this was a TODO in the original exp/vanityv2.sql, never actually done -- ported as-is, single-dash typo fixed so it parses as a comment instead of a syntax error.)
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/vanityv2.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
