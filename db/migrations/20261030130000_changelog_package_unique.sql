-- +goose Up
-- +goose StatementBegin
ALTER TABLE changelogs DROP CONSTRAINT IF EXISTS changelogs_project_version_key;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS changelogs_project_package_version_key ON changelogs (project, COALESCE(package, ''), version);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS changelogs_project_package_version_key;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE changelogs ADD CONSTRAINT changelogs_project_version_key UNIQUE (project, version);
-- +goose StatementEnd
