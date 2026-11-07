-- +goose Up
-- +goose StatementBegin
ALTER TABLE staffpanel__authchain ADD COLUMN ip text;
ALTER TABLE staffpanel__authchain ADD COLUMN user_agent text;
CREATE INDEX staffpanel__authchain_user_id_idx ON staffpanel__authchain (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX staffpanel__authchain_user_id_idx;
ALTER TABLE staffpanel__authchain DROP COLUMN user_agent;
ALTER TABLE staffpanel__authchain DROP COLUMN ip;
-- +goose StatementEnd
