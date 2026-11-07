-- +goose Up
-- +goose StatementBegin
ALTER TABLE auto_vote_subscriptions
    ADD COLUMN extra_slots integer DEFAULT 0 NOT NULL CHECK (extra_slots >= 0),
    ADD COLUMN billing_interval text DEFAULT 'month' NOT NULL CHECK (billing_interval IN ('month', 'year'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE auto_vote_subscriptions DROP COLUMN extra_slots, DROP COLUMN billing_interval;
-- +goose StatementEnd
