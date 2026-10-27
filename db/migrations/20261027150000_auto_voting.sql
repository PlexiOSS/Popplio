-- +goose Up
-- +goose StatementBegin
ALTER TABLE entity_votes ADD COLUMN auto boolean DEFAULT false NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE bots ADD COLUMN auto_votes_opt_out boolean DEFAULT false NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN auto_votes_opt_out boolean DEFAULT false NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE auto_vote_subscriptions (
    user_id text NOT NULL PRIMARY KEY,
    stripe_customer_id text NOT NULL,
    stripe_subscription_id text NOT NULL UNIQUE,
    status text NOT NULL,
    current_period_end timestamp with time zone,
    cancel_at_period_end boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE auto_vote_targets (
    user_id text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    days smallint[] DEFAULT '{}'::smallint[] NOT NULL,
    start_minute smallint DEFAULT 0 NOT NULL,
    end_minute smallint DEFAULT 1440 NOT NULL,
    timezone text DEFAULT 'UTC'::text NOT NULL,
    last_voted_at timestamp with time zone,
    last_skip_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id, target_type, target_id),
    CONSTRAINT auto_vote_targets_type_check CHECK (target_type IN ('bot', 'server')),
    CONSTRAINT auto_vote_targets_minutes_check CHECK (start_minute >= 0 AND end_minute <= 1440 AND start_minute < end_minute)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE auto_vote_targets;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE auto_vote_subscriptions;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN auto_votes_opt_out;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE bots DROP COLUMN auto_votes_opt_out;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE entity_votes DROP COLUMN auto;
-- +goose StatementEnd
