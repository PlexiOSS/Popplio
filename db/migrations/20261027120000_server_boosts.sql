-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers
    ADD COLUMN premium_tier smallint DEFAULT 0 NOT NULL,
    ADD COLUMN boost_count integer DEFAULT 0 NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE servers
    DROP COLUMN premium_tier,
    DROP COLUMN boost_count;
-- +goose StatementEnd
