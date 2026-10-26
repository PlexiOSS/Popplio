-- +goose Up
-- +goose StatementBegin
ALTER TABLE badges
    ADD COLUMN auto_rule jsonb,
    ADD COLUMN linked_role boolean DEFAULT false NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE badges SET auto_rule = '{"type": "partner"}'::jsonb WHERE id = 'partner' AND auto_rule IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE badges SET auto_rule = '{"type": "listed_days", "value": 730}'::jsonb WHERE id = 'veteran' AND auto_rule IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE user_role_connections (
    user_id text NOT NULL,
    access_token bytea NOT NULL,
    refresh_token bytea NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    last_metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY user_role_connections
    ADD CONSTRAINT user_role_connections_pkey PRIMARY KEY (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_role_connections;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE badges DROP COLUMN linked_role, DROP COLUMN auto_rule;
-- +goose StatementEnd
