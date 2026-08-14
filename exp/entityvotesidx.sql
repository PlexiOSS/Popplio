-- CONCURRENTLY avoids the write-blocking lock a plain CREATE INDEX takes
-- for the whole build — entity_votes is a high-write table (every vote is
-- an insert), so that lock would visibly stall voting for the duration.
--
-- Must be run as its own statement outside a transaction block — psql -1,
-- or wrapping this file in BEGIN/COMMIT, will make CONCURRENTLY fail
-- outright ("cannot run inside a transaction block"). Run it as:
--   psql "$DATABASE_URL" -f exp/entityvotesidx.sql
-- not psql -1, and not from any tool that auto-wraps files in a transaction.
create index concurrently if not exists entity_votes_target_created_idx on entity_votes(target_type, target_id, created_at) where void = false;
