-- +goose NO TRANSACTION
-- +goose Up
-- The Search List endpoint (routes/list/endpoints/search_list) matches
-- `short @@ $query` on bots and `name ILIKE '%...%' OR name @@ $query OR
-- short @@ $query` on servers (sql/bots.tmpl, sql/servers.tmpl). None of
-- these have a supporting index, so every search request does a full
-- sequential scan across every approved/certified bot and server,
-- recomputing to_tsvector() per row on every call.
--
-- Indexes below use `to_tsvector('english', col)` (config as a literal)
-- rather than the single-arg `to_tsvector(col)` form: the single-arg form
-- resolves its config via get_current_ts_config(), which is STABLE, not
-- IMMUTABLE, and Postgres refuses to index a STABLE expression ("functions
-- in index expression must be marked IMMUTABLE"). The `short @@ $query`
-- text @@ text operator used in the .tmpl files also resolves its config
-- via get_current_ts_config() at query time, so this index only gets used
-- when that resolves to 'english' — true for any deployment that hasn't
-- overridden the default_text_search_config setting (default is
-- pg_catalog.english).
--
-- CONCURRENTLY avoids the write-blocking lock a plain CREATE INDEX takes
-- for the whole build. Must be run as its own statement outside a
-- transaction block — psql -1, or wrapping this file in BEGIN/COMMIT,
-- will make CONCURRENTLY fail outright ("cannot run inside a transaction
-- block"). Run it as:
--   psql "$DATABASE_URL" -f exp/search_gin_idx.sql
-- not psql -1, and not from any tool that auto-wraps files in a
-- transaction.
--
-- Below: each statement gets its own StatementBegin/StatementEnd block.
-- goose sends everything inside one block as a single query string, and
-- Postgres implicitly wraps a *multi-statement* query string in a
-- transaction even without an explicit BEGIN -- which breaks CONCURRENTLY
-- just like an explicit transaction would. One statement per block avoids
-- that.

-- +goose StatementBegin
create extension if not exists pg_trgm;
-- +goose StatementEnd

-- Full-text match support for `short @@ $query` (bots) / `name @@ $query`,
-- `short @@ $query` (servers)
-- +goose StatementBegin
create index concurrently if not exists bots_short_fts_idx on bots using gin (to_tsvector('english', short));
-- +goose StatementEnd

-- +goose StatementBegin
create index concurrently if not exists servers_name_fts_idx on servers using gin (to_tsvector('english', name));
-- +goose StatementEnd

-- +goose StatementBegin
create index concurrently if not exists servers_short_fts_idx on servers using gin (to_tsvector('english', short));
-- +goose StatementEnd

-- Trigram support for `name ILIKE '%...%'` on servers (bots has no ILIKE
-- match on its own columns — only on the joined platform-users table,
-- which popplio doesn't own)
-- +goose StatementBegin
create index concurrently if not exists servers_name_trgm_idx on servers using gin (name gin_trgm_ops);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/search_gin_idx.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
