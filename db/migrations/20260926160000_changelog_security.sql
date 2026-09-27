-- +goose Up
-- +goose StatementBegin
ALTER TABLE changelogs ADD COLUMN security text[] DEFAULT '{}'::text[] NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE changelogs DROP COLUMN security;
-- +goose StatementEnd
