-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    source text NOT NULL,
    actor_type text NOT NULL,
    actor_id text,
    action text NOT NULL,
    target_type text,
    target_id text,
    status integer,
    ip text,
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT audit_events_source_check CHECK (source IN ('api', 'panel', 'system', 'stripe', 'paypal'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX audit_events_actor_idx ON audit_events (actor_id, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX audit_events_target_idx ON audit_events (target_id, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX audit_events_action_idx ON audit_events (action text_pattern_ops, id DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX audit_events_created_idx ON audit_events (created_at);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE payment_fulfillments ADD COLUMN revoked_at timestamp with time zone;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE payment_fulfillments ADD COLUMN revoke_reason text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE payment_fulfillments DROP COLUMN revoke_reason;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE payment_fulfillments DROP COLUMN revoked_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE audit_events;
-- +goose StatementEnd
