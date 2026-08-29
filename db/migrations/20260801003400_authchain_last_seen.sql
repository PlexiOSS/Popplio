-- +goose Up
-- +goose StatementBegin
-- Adds sliding-expiration support to staffpanel__authchain.
--
-- Before this, an active staff-panel session hard-expired exactly 1 hour
-- after it was created (arcadia/impls/auth.go), with no way to extend it —
-- staying actively logged in and working for over an hour still meant being
-- kicked back to login. `last_seen_at` is a separate column from
-- `created_at` (which keeps meaning what it always meant: when the row was
-- first created) that CheckAuth bumps to NOW() on every successful
-- authenticated request for an active session. The 1-hour prune now runs
-- against `last_seen_at` instead, so a session only expires after an hour
-- of no activity at all, not an hour of wall-clock time.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/authchain_last_seen.sql
--
-- Safe to re-run: guarded on the column already existing.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'staffpanel__authchain' AND column_name = 'last_seen_at'
    ) THEN
        ALTER TABLE staffpanel__authchain ADD COLUMN last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
        RAISE NOTICE 'added staffpanel__authchain.last_seen_at';
    ELSE
        RAISE NOTICE 'staffpanel__authchain.last_seen_at already exists, skipping';
    END IF;
END
$$;

-- =====================================================================
-- ROLLBACK (commented out; run by hand only if this needs to be fully undone)
--
-- BEGIN;
-- ALTER TABLE staffpanel__authchain DROP COLUMN IF EXISTS last_seen_at;
-- COMMIT;
-- =====================================================================
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/rewrite/authchain_last_seen.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
