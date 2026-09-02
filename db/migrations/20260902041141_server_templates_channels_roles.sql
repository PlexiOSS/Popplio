-- +goose Up
-- +goose StatementBegin
ALTER TABLE server_templates
    ADD COLUMN channels jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN roles jsonb NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE server_templates DROP COLUMN channels, DROP COLUMN roles;
-- +goose StatementEnd
