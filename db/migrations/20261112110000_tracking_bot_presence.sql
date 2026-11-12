-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN tracking_bot_missing_since timestamptz;
ALTER TABLE servers ADD COLUMN tracking_bot_hidden_state text;
ALTER TABLE servers ADD COLUMN tracking_bot_notice smallint NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE servers SET state = tracking_bot_hidden_state WHERE tracking_bot_hidden_state IS NOT NULL;
ALTER TABLE servers DROP COLUMN tracking_bot_notice;
ALTER TABLE servers DROP COLUMN tracking_bot_hidden_state;
ALTER TABLE servers DROP COLUMN tracking_bot_missing_since;
-- +goose StatementEnd
