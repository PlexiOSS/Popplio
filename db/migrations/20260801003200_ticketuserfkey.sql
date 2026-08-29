-- +goose Up
-- +goose StatementBegin
-- tickets.user_id has never had a real foreign key constraint to users(user_id),
-- so the data-export/deletion task walker (routes/users/endpoints/create_data_task)
-- silently skips it: the walk only follows tables with a real pg_constraint edge
-- into an already-registered root table ("users"/"teams" in tableops.go).
--
-- Added NOT VALID: 4 existing tickets reference a user_id that no longer has a
-- row in `users` (accounts deleted before this constraint existed, one ticket
-- literally says "my old discord account got deleted"). NOT VALID enforces
-- the constraint for all new/updated rows going forward without requiring
-- those legacy rows to be deleted or have their ownership nulled out. It's
-- still picked up by the DDR walker's pg_constraint introspection (validity
-- isn't part of that query), so tickets become exportable/deletable for every
-- current user immediately.
--
-- Guarded on the port: genesis_foundational_tables.sql already captures
-- this constraint since prod already has it.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tickets_user_id_fkey' AND conrelid = 'tickets'::regclass
    ) THEN
        ALTER TABLE tickets
            ADD CONSTRAINT tickets_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES users(user_id)
            ON UPDATE CASCADE ON DELETE CASCADE
            NOT VALID;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/ticketuserfkey.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
