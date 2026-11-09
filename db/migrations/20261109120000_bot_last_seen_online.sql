-- +goose Up
-- +goose StatementBegin
ALTER TABLE bots ADD COLUMN last_seen_online timestamp with time zone;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bots DROP COLUMN last_seen_online;
-- +goose StatementEnd
