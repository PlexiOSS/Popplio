-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN vote_channel_id text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN vote_channel_id;
-- +goose StatementEnd
