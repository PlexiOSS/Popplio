-- Adds "keel" as a third valid changelog project, alongside popplio and
-- omniplex -- Keel (github.com/PlexiOSS/Keel) is the shared library several
-- other projects in this org depend on and now gets its own curated release
-- history too.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/changelogs_add_keel.sql
--
-- Safe to re-run: guarded on the constraint already allowing 'keel'.

\set ON_ERROR_STOP on

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'changelogs_project_check'
          AND pg_get_constraintdef(oid) LIKE '%''keel''%'
    ) THEN
        ALTER TABLE changelogs DROP CONSTRAINT IF EXISTS changelogs_project_check;
        ALTER TABLE changelogs ADD CONSTRAINT changelogs_project_check CHECK (project IN ('popplio', 'omniplex', 'keel'));
        RAISE NOTICE 'changelogs_project_check now allows keel';
    ELSE
        RAISE NOTICE 'changelogs_project_check already allows keel, skipping';
    END IF;
END
$$;

COMMIT;

-- =====================================================================
-- ROLLBACK (commented out; run by hand only if this needs to be fully undone)
--
-- BEGIN;
-- ALTER TABLE changelogs DROP CONSTRAINT IF EXISTS changelogs_project_check;
-- ALTER TABLE changelogs ADD CONSTRAINT changelogs_project_check CHECK (project IN ('popplio', 'omniplex'));
-- COMMIT;
-- =====================================================================
