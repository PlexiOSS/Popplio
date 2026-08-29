-- +goose Up
-- +goose StatementBegin
alter table team_members add column if not exists itag UUID PRIMARY KEY DEFAULT uuid_generate_v4();

alter table team_members add column if not exists mentionable boolean not null default true;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/teamsmentionable.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
