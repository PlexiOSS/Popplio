-- +goose Up
-- +goose StatementBegin
ALTER TABLE changelogs ADD COLUMN package text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE changelogs ADD CONSTRAINT changelogs_package_check CHECK (package IS NULL OR package ~ '^[a-z0-9][a-z0-9-]{0,63}$');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE changelogs DROP CONSTRAINT changelogs_package_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE changelogs DROP COLUMN package;
-- +goose StatementEnd
