-- Adds a `fixed` column to changelogs, matching Keep a Changelog's standard
-- Added/Changed/Fixed/Removed categories (this project's own CHANGELOG.md
-- already follows that format) -- the curated changelog system originally
-- shipped with only added/updated/removed.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/changelogs_add_fixed.sql
--
-- Safe to re-run: guarded on the column already existing.

\set ON_ERROR_STOP on

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'fixed'
    ) THEN
        ALTER TABLE changelogs ADD COLUMN fixed TEXT[] NOT NULL DEFAULT '{}';
        RAISE NOTICE 'added changelogs.fixed';
    ELSE
        RAISE NOTICE 'changelogs.fixed already exists, skipping';
    END IF;
END
$$;

COMMIT;

-- =====================================================================
-- ROLLBACK (commented out; run by hand only if this needs to be fully undone)
--
-- BEGIN;
-- ALTER TABLE changelogs DROP COLUMN IF EXISTS fixed;
-- COMMIT;
-- =====================================================================
