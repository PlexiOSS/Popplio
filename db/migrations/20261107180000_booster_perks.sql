-- +goose Up
-- +goose StatementBegin
CREATE TABLE booster_perks (
    user_id text NOT NULL,
    boosting boolean DEFAULT false NOT NULL,
    boosting_since timestamp with time zone,
    target_type text,
    target_id text,
    last_error text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT booster_perks_pkey PRIMARY KEY (user_id),
    CONSTRAINT booster_perks_target_type_check CHECK (target_type IS NULL OR target_type IN ('bot', 'server')),
    CONSTRAINT booster_perks_target_check CHECK ((target_type IS NULL) = (target_id IS NULL))
);

ALTER TABLE payment_fulfillments DROP CONSTRAINT payment_fulfillments_provider_check;
ALTER TABLE payment_fulfillments ADD CONSTRAINT payment_fulfillments_provider_check CHECK (provider = ANY (ARRAY['stripe'::text, 'paypal'::text, 'booster'::text]));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE booster_perks;
DELETE FROM payment_fulfillments WHERE provider = 'booster';
ALTER TABLE payment_fulfillments DROP CONSTRAINT payment_fulfillments_provider_check;
ALTER TABLE payment_fulfillments ADD CONSTRAINT payment_fulfillments_provider_check CHECK (provider = ANY (ARRAY['stripe'::text, 'paypal'::text]));
-- +goose StatementEnd
