-- Lets a server owner self-report stats via a server-scoped API token
-- (POST /servers/stats), the same way bots already can via POST /bots/stats.
-- stats_self_managed marks a server as actively self-reporting, so the
-- periodic syncServerMeta task (infernoplex/tasks/serversync.go) knows to
-- stop overwriting total_members/online_members for it — otherwise the two
-- update paths would just fight each other every 30 minutes.
alter table servers add column stats_self_managed boolean not null default false;
alter table servers add column last_stats_post timestamptz;
