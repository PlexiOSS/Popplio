-- Backs the reviewer-facing NSFW compliance check on the review queue: does
-- a server actually gate its NSFW content behind Discord's own age-restricted
-- channel flag, and/or does it match Discord's own guild-level NSFW
-- classification? Previously a reviewer had to join the server and look
-- around by hand to answer "Server: NSFW Content Not Gated" these columns
-- are synced by Infernoplex's periodic serversync task (syncServerMeta) the
-- same way avatar/member counts already are.
--
-- discord_nsfw_level mirrors Discord's own guild.nsfw_level (0=Default,
-- 1=Explicit, 2=Safe, 3=AgeRestricted). nsfw_channel_count is how many of the
-- guild's channels currently have Discord's own age-restricted flag set.
--
-- Run it as:
--   psql "$DATABASE_URL" -f exp/server_nsfw_compliance.sql

alter table servers add column if not exists discord_nsfw_level smallint not null default 0;
alter table servers add column if not exists nsfw_channel_count integer not null default 0;
