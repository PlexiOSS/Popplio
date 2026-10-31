-- +goose Up
-- +goose StatementBegin
CREATE TABLE promo_codes (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    code text NOT NULL UNIQUE,
    description text DEFAULT '' NOT NULL,
    products text[] NOT NULL,
    percent_off integer,
    amount_off_cents integer,
    duration text NOT NULL,
    duration_months integer,
    max_redemptions integer,
    expires_at timestamp with time zone,
    once_per_user boolean DEFAULT false NOT NULL,
    new_customers_only boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    stripe_coupon_id text NOT NULL,
    stripe_promotion_code_id text NOT NULL,
    created_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT promo_codes_code_check CHECK (code ~ '^[A-Z0-9_-]{3,40}$'),
    CONSTRAINT promo_codes_products_check CHECK (cardinality(products) > 0 AND products <@ ARRAY['premium', 'autovote']::text[]),
    CONSTRAINT promo_codes_discount_check CHECK ((percent_off IS NULL) <> (amount_off_cents IS NULL)),
    CONSTRAINT promo_codes_percent_check CHECK (percent_off IS NULL OR (percent_off > 0 AND percent_off <= 100)),
    CONSTRAINT promo_codes_amount_check CHECK (amount_off_cents IS NULL OR amount_off_cents > 0),
    CONSTRAINT promo_codes_duration_check CHECK (duration IN ('once', 'repeating', 'forever')),
    CONSTRAINT promo_codes_duration_months_check CHECK ((duration = 'repeating') = (duration_months IS NOT NULL AND duration_months > 0)),
    CONSTRAINT promo_codes_max_redemptions_check CHECK (max_redemptions IS NULL OR max_redemptions > 0)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE promo_code_redemptions (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    promo_code_id uuid NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id text NOT NULL,
    product text NOT NULL,
    stripe_session_id text NOT NULL UNIQUE,
    status text DEFAULT 'pending' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT promo_code_redemptions_product_check CHECK (product IN ('premium', 'autovote')),
    CONSTRAINT promo_code_redemptions_status_check CHECK (status IN ('pending', 'completed', 'abandoned'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX promo_code_redemptions_user_idx ON promo_code_redemptions (promo_code_id, user_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE promo_code_redemptions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE promo_codes;
-- +goose StatementEnd
