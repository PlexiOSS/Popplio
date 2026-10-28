-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_stats_daily (
    target_type text NOT NULL,
    target_id text NOT NULL,
    day date NOT NULL,
    clicks integer DEFAULT 0 NOT NULL,
    invite_clicks integer DEFAULT 0 NOT NULL,
    servers integer DEFAULT 0 NOT NULL,
    users integer DEFAULT 0 NOT NULL,
    total_members integer DEFAULT 0 NOT NULL,
    online_members integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (target_type, target_id, day)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX entity_votes_target_all_created_idx ON entity_votes (target_type, target_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX entity_votes_target_all_created_idx;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE entity_stats_daily;
-- +goose StatementEnd
