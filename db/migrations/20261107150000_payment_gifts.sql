-- +goose Up
-- +goose StatementBegin
CREATE TABLE payment_gifts (
    provider text NOT NULL,
    external_id text NOT NULL,
    gifter_id text NOT NULL,
    message text DEFAULT '' NOT NULL,
    anonymous boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (provider, external_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE payment_gifts;
-- +goose StatementEnd
