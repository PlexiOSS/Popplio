-- Backs the periodic moderation_scan background task (popplio/bgtasks), which
-- re-runs OpenAI moderation against every bot's/server's short/long
-- description on a rolling basis — not just at submission time — and files
-- a report for staff when something's newly flagged. This is the cursor
-- that lets the task pick up where it left off instead of rescanning the
-- entire list every run: NULL means "never checked" (pre-existing entries
-- from before moderation checks existed, or before this task existed at
-- all), anything else is when it was last checked.
--
-- Run it as:
--   psql "$DATABASE_URL" -f exp/moderation_scan_columns.sql

alter table bots add column if not exists moderation_checked_at timestamptz;
alter table servers add column if not exists moderation_checked_at timestamptz;
