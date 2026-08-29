-- +goose Up
-- +goose StatementBegin
-- Adds servers.avatar: servers currently have no icon source at all for
-- listings or the detail page. The old cache-server subsystem (removed via
-- exp/remove_cache_servers.sql) used to synthesize this from its own CDN
-- cache; nothing replaced it. Populated once at Add Server time from the
-- invite resolution already done there, and kept fresh afterward by
-- Infernoplex's serversync task (reads from its gateway cache, so it's
-- naturally skipped for any server the bot isn't currently in).
--
--   USAGE
--     psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f exp/add_servers_avatar.sql

ALTER TABLE servers ADD COLUMN IF NOT EXISTS avatar TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/add_servers_avatar.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
