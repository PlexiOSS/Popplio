-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_views_daily (
    target_type text NOT NULL,
    target_id text NOT NULL,
    day date NOT NULL,
    views integer DEFAULT 0 NOT NULL,
    CONSTRAINT entity_views_daily_pkey PRIMARY KEY (target_type, target_id, day),
    CONSTRAINT entity_views_daily_target_type_check CHECK (target_type IN ('bot', 'server'))
);

CREATE INDEX entity_views_daily_day_idx ON entity_views_daily (day);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entity_views_daily;
-- +goose StatementEnd
