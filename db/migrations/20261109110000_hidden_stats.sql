-- +goose Up
-- +goose StatementBegin
ALTER TABLE bots ADD COLUMN hidden_stats text[] DEFAULT '{}'::text[] NOT NULL;
ALTER TABLE servers ADD COLUMN hidden_stats text[] DEFAULT '{}'::text[] NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bots DROP COLUMN hidden_stats;
ALTER TABLE servers DROP COLUMN hidden_stats;
-- +goose StatementEnd
