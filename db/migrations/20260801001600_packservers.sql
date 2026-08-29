-- +goose Up
-- +goose StatementBegin
alter table packs add column if not exists servers text[] not null default '{}';
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/packservers.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
