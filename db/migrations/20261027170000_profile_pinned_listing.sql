-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN pinned_target_type text,
    ADD COLUMN pinned_target_id text,
    ADD CONSTRAINT users_pinned_target_check CHECK (
        (pinned_target_type IS NULL AND pinned_target_id IS NULL)
        OR (pinned_target_type IN ('bot', 'server') AND pinned_target_id IS NOT NULL)
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP CONSTRAINT users_pinned_target_check,
    DROP COLUMN pinned_target_type,
    DROP COLUMN pinned_target_id;
-- +goose StatementEnd
