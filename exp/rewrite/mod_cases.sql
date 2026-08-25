-- Adds mod_cases: a durable record of every moderation action taken by
-- Arcadia (kick, ban, timeout, warn, purge, lock/unlock, and auto-mod
-- actions), backing the /modlogs lookup command.
--
-- Before this, logModeration only posted a Discord embed to the mod-logs
-- channel -- nothing was queryable. This table is additive: the Discord post
-- stays exactly as it is (it's the live-incident view), this is the
-- searchable history behind it.
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/rewrite/mod_cases.sql
--
-- Re-running this script is a no-op (every statement is guarded).

\set ON_ERROR_STOP on

BEGIN;

CREATE TABLE IF NOT EXISTS mod_cases (
    case_id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guild_id     TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    -- moderator_id is the acting staff member's user id, or the bot's own
    -- application user id for an automated auto-mod action (see arcadia's
    -- automod.go) -- there is no separate "system" sentinel, the bot is just
    -- another actor with its own snowflake, which keeps this column a plain
    -- foreign-key-shaped user id rather than a nullable/tagged union.
    moderator_id TEXT NOT NULL,
    action       TEXT NOT NULL CHECK (action IN ('kick', 'ban', 'timeout', 'warn', 'purge', 'lock', 'unlock', 'automod')),
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE mod_cases IS
    'Durable moderation history backing /modlogs. Written by arcadia/bot on every moderation action, including automod. See exp/rewrite/mod_cases.sql.';

-- The lookup /modlogs runs: most-recent-first by user within a guild.
CREATE INDEX IF NOT EXISTS mod_cases_guild_user_idx ON mod_cases (guild_id, user_id, created_at DESC);

COMMIT;

-- =====================================================================
-- ROLLBACK (commented out; run by hand only if this needs to be fully undone)
--
-- BEGIN;
-- DROP TABLE IF EXISTS mod_cases;
-- COMMIT;
-- =====================================================================
