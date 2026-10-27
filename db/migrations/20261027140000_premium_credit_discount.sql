-- +goose Up
-- +goose StatementBegin
CREATE TABLE payment_credit_holds (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    provider text NOT NULL,
    external_id text NOT NULL,
    user_id text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    credits integer NOT NULL,
    discount_cents integer NOT NULL,
    state text DEFAULT 'held' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    CONSTRAINT payment_credit_holds_credits_check CHECK (credits > 0),
    CONSTRAINT payment_credit_holds_state_check CHECK (state IN ('held', 'spent', 'refunded')),
    CONSTRAINT payment_credit_holds_provider_external_key UNIQUE (provider, external_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX payment_credit_holds_held_idx ON payment_credit_holds (created_at) WHERE state = 'held';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shop_items SET cents = 600 WHERE id = 'premium/bronze' AND cents < 600;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shop_items SET cents = 150 WHERE id = 'premium-days-7' AND cents < 150;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE payment_credit_holds;
-- +goose StatementEnd
