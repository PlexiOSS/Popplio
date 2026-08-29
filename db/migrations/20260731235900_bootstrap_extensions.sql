-- +goose Up
-- +goose StatementBegin
-- None of these was ever captured in any exp/*.sql file -- all three were
-- enabled on prod ad hoc, outside any tracked migration. Discovered by
-- actually running every ported migration against a fresh database
-- (vanityv2's CITEXT column failed with "type citext does not exist";
-- separately, the genesis_foundational_tables migration's gin_trgm_ops
-- indexes, e.g. servers_name_trgm_idx, failed because pg_trgm wasn't
-- created yet -- it's originally created in search_gin_idx.sql, but that
-- migration is dated *after* genesis_foundational_tables, so pg_trgm has
-- to exist before then too. search_gin_idx.sql's own
-- "create extension if not exists pg_trgm" is left in place -- IF NOT
-- EXISTS there makes the duplication harmless).
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- +goose StatementEnd

-- +goose Down
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS "uuid-ossp";
