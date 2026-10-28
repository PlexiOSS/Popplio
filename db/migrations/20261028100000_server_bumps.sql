-- +goose Up
-- +goose StatementBegin
ALTER TABLE servers ADD COLUMN bumped_at timestamp with time zone;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE server_bumps (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    server_id text NOT NULL REFERENCES servers (server_id) ON DELETE CASCADE,
    user_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX server_bumps_server_created_idx ON server_bumps (server_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX servers_bumped_at_idx ON servers (bumped_at DESC) WHERE bumped_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE server_bumps;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN bumped_at;
-- +goose StatementEnd
