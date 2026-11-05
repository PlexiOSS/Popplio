-- +goose Up
-- +goose StatementBegin
ALTER TABLE payment_fulfillments ADD COLUMN premium_hours integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE payment_fulfillments DROP COLUMN premium_hours;
-- +goose StatementEnd
