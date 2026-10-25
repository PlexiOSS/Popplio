-- +goose Up
-- +goose StatementBegin
CREATE TABLE payment_fulfillments (
    provider text NOT NULL,
    external_id text NOT NULL,
    user_id text NOT NULL,
    product_id text NOT NULL,
    for_type text NOT NULL,
    for_id text NOT NULL,
    fulfilled_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT payment_fulfillments_provider_check CHECK (provider = ANY (ARRAY['stripe'::text, 'paypal'::text]))
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY payment_fulfillments
    ADD CONSTRAINT payment_fulfillments_pkey PRIMARY KEY (provider, external_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE payment_fulfillments;
-- +goose StatementEnd
