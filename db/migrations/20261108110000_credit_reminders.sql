-- +goose Up
-- +goose StatementBegin
CREATE TABLE credit_reminders (
    user_id text NOT NULL,
    period text NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT credit_reminders_pkey PRIMARY KEY (user_id, period)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE credit_reminders;
-- +goose StatementEnd
