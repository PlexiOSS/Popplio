-- +goose Up
-- +goose StatementBegin
ALTER TABLE automated_vote_resets
    ADD COLUMN period date,
    ADD COLUMN kind text DEFAULT 'automatic' NOT NULL,
    ADD COLUMN target_type text,
    ADD COLUMN triggered_by text,
    ADD CONSTRAINT automated_vote_resets_kind_check CHECK (kind IN ('automatic', 'manual'));

UPDATE automated_vote_resets SET period = date_trunc('month', created_at AT TIME ZONE 'UTC')::date;

CREATE INDEX automated_vote_resets_period_idx ON automated_vote_resets (period, kind);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM automated_vote_resets WHERE kind = 'manual';
ALTER TABLE automated_vote_resets
    DROP COLUMN period,
    DROP COLUMN kind,
    DROP COLUMN target_type,
    DROP COLUMN triggered_by;
-- +goose StatementEnd
