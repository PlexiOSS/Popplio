-- Backs OpenAI moderation checks (popplio/moderation), run against a bot's
-- or server's short/long description at submission time. This is a signal
-- for reviewers, not a verdict: nothing reads these columns to auto-approve
-- or auto-deny, they just surface on the review queue.
--
-- Run it as:
--   psql "$DATABASE_URL" -f exp/moderation_columns.sql

alter table bots add column if not exists moderation_flagged boolean not null default false;
alter table bots add column if not exists moderation_categories text[] not null default '{}';

alter table servers add column if not exists moderation_flagged boolean not null default false;
alter table servers add column if not exists moderation_categories text[] not null default '{}';
