-- +goose Up
-- +goose StatementBegin
ALTER TABLE server_templates
    ADD COLUMN icon_version text,
    ADD COLUMN source_guild_id text,
    ADD COLUMN source_updated_at timestamp with time zone,
    ADD COLUMN out_of_date boolean DEFAULT false NOT NULL,
    ADD COLUMN unavailable boolean DEFAULT false NOT NULL,
    ADD COLUMN synced_at timestamp with time zone;
UPDATE server_templates SET synced_at = updated_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE server_templates
    DROP COLUMN icon_version,
    DROP COLUMN source_guild_id,
    DROP COLUMN source_updated_at,
    DROP COLUMN out_of_date,
    DROP COLUMN unavailable,
    DROP COLUMN synced_at;
-- +goose StatementEnd
