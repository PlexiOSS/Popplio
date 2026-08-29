-- +goose Up
-- +goose StatementBegin
alter table servers add column if not exists show_emojis boolean not null default false;
alter table servers add column if not exists emojis jsonb not null default '[]'::jsonb;
alter table servers add column if not exists stickers jsonb not null default '[]'::jsonb;
alter table servers add column if not exists emojis_synced_at timestamptz;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/serveremojis.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
