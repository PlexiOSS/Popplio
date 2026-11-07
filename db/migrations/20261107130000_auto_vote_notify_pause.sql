-- +goose Up
-- +goose StatementBegin
ALTER TABLE auto_vote_targets ADD COLUMN paused boolean NOT NULL DEFAULT false;
CREATE TABLE auto_vote_settings (
    user_id text PRIMARY KEY,
    notify text NOT NULL DEFAULT 'off',
    last_summary_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auto_vote_settings_notify_check CHECK (notify IN ('off', 'each', 'daily'))
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE auto_vote_settings;
ALTER TABLE auto_vote_targets DROP COLUMN paused;
-- +goose StatementEnd
