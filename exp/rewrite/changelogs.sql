-- Wires up the `changelogs` table: a curated, staff-authored release history
-- for BOTH Popplio (the API) and Omniplex (the website), replacing the site's
-- previous approach of pulling raw GitHub release notes live at request time.
--
-- exp/changelogsv2.sql defined an earlier draft of this table (bare
-- `version TEXT PRIMARY KEY`, no `project` column) that nothing in the
-- codebase ever read or wrote -- the Arcadia panel's UpdateChangelog action
-- was a hard stub returning 403. A single `version` primary key can't
-- actually support two independent projects (Popplio and Omniplex could
-- each reach the same version string), so this script creates the real
-- schema keyed on (project, version) instead, and migrates forward in place
-- if the old draft's bare table already exists in prod.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/changelogs.sql
--
-- Safe to re-run: every statement is guarded.

\set ON_ERROR_STOP on

BEGIN;

CREATE TABLE IF NOT EXISTS changelogs (
    itag               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project             TEXT NOT NULL CHECK (project IN ('popplio', 'omniplex')),
    version             TEXT NOT NULL,
    added               TEXT[] NOT NULL DEFAULT '{}',
    updated             TEXT[] NOT NULL DEFAULT '{}',
    removed             TEXT[] NOT NULL DEFAULT '{}',
    extra_description   TEXT NOT NULL DEFAULT '',
    prerelease          BOOLEAN NOT NULL DEFAULT FALSE,
    published           BOOLEAN NOT NULL DEFAULT FALSE,
    created_by          TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project, version)
);

-- Migrate an already-existing bare table from the old exp/changelogsv2.sql
-- draft forward to the current shape. Every branch is guarded on whether the
-- column/constraint already exists, so this is a no-op on a fresh table
-- created by the CREATE TABLE above, and a no-op on a second run.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'changelogs') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'itag') THEN
            ALTER TABLE changelogs ADD COLUMN itag UUID DEFAULT gen_random_uuid();
            UPDATE changelogs SET itag = gen_random_uuid() WHERE itag IS NULL;
            ALTER TABLE changelogs ALTER COLUMN itag SET NOT NULL;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'project') THEN
            -- The old draft predates Omniplex being tracked at all, so every
            -- existing row is a Popplio entry.
            ALTER TABLE changelogs ADD COLUMN project TEXT DEFAULT 'popplio';
            UPDATE changelogs SET project = 'popplio' WHERE project IS NULL;
            ALTER TABLE changelogs ALTER COLUMN project SET NOT NULL;
            ALTER TABLE changelogs ADD CONSTRAINT changelogs_project_check CHECK (project IN ('popplio', 'omniplex'));
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'published') THEN
            ALTER TABLE changelogs ADD COLUMN published BOOLEAN NOT NULL DEFAULT FALSE;
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'created_by') THEN
            ALTER TABLE changelogs ADD COLUMN created_by TEXT DEFAULT 'unknown';
            ALTER TABLE changelogs ALTER COLUMN created_by SET NOT NULL;
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'changelogs' AND column_name = 'github_html') THEN
            ALTER TABLE changelogs DROP COLUMN github_html;
        END IF;

        -- The old draft's PK was `version` alone; replace it with the
        -- surrogate `itag` PK plus a (project, version) uniqueness
        -- constraint now that project exists. Postgres names a table's PK
        -- constraint "<table>_pkey" by default, so the old version-only PK
        -- is ALSO named "changelogs_pkey" -- the check below has to look at
        -- which COLUMN the pkey constraint is actually on, not its name.
        IF EXISTS (
            SELECT 1
            FROM information_schema.key_column_usage
            WHERE table_schema = 'public' AND table_name = 'changelogs'
              AND constraint_name = 'changelogs_pkey' AND column_name = 'version'
        ) THEN
            ALTER TABLE changelogs DROP CONSTRAINT changelogs_pkey;
            ALTER TABLE changelogs ADD PRIMARY KEY (itag);
        END IF;

        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE table_schema = 'public' AND table_name = 'changelogs' AND constraint_name = 'changelogs_project_version_key') THEN
            ALTER TABLE changelogs ADD CONSTRAINT changelogs_project_version_key UNIQUE (project, version);
        END IF;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS changelogs_project_published_idx ON changelogs (project, published, created_at DESC);

COMMIT;

-- =====================================================================
-- ROLLBACK (commented out; run by hand only if this needs to be fully undone)
--
-- BEGIN;
-- DROP TABLE IF EXISTS changelogs;
-- COMMIT;
-- =====================================================================
