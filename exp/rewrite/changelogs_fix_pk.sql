-- Follow-up to exp/rewrite/changelogs.sql.
--
-- That script's PK-swap step checked for a primary key constraint named
-- anything OTHER than "changelogs_pkey", intending to catch the old draft's
-- PK on `version` alone and replace it with one on the new surrogate `itag`
-- column. But Postgres's default naming convention already names a table's
-- primary key "<table>_pkey" -- so the old `version`-only PK WAS named
-- "changelogs_pkey", the condition was always false, and the swap never
-- ran. Confirmed via `\d changelogs` against prod: the PK was still on
-- `version` alone after running that script.
--
-- This corrects it. Guarded so it's a no-op if changelogs_pkey is already on
-- itag (e.g. run against a fresh table created directly by changelogs.sql's
-- CREATE TABLE, which never has this bug).
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/changelogs_fix_pk.sql

\set ON_ERROR_STOP on

BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.key_column_usage
        WHERE table_schema = 'public' AND table_name = 'changelogs'
          AND constraint_name = 'changelogs_pkey' AND column_name = 'version'
    ) THEN
        ALTER TABLE changelogs DROP CONSTRAINT changelogs_pkey;
        ALTER TABLE changelogs ADD PRIMARY KEY (itag);
        RAISE NOTICE 'changelogs primary key moved from version to itag';
    ELSE
        RAISE NOTICE 'changelogs primary key is already on itag, nothing to do';
    END IF;
END
$$;

COMMIT;
