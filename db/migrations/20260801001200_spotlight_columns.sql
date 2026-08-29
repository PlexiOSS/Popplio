-- +goose Up
-- +goose StatementBegin
alter table bots add column if not exists spotlighted_until timestamptz;
alter table servers add column if not exists spotlighted_until timestamptz;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/spotlight_columns.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
