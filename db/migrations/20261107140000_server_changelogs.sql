-- +goose Up
-- +goose StatementBegin
CREATE TABLE server_changelogs (
    id uuid DEFAULT uuid_generate_v4() PRIMARY KEY,
    server_id text NOT NULL REFERENCES servers(server_id) ON DELETE CASCADE,
    title text NOT NULL,
    content text DEFAULT '' NOT NULL,
    version text DEFAULT '' NOT NULL,
    created_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
CREATE INDEX server_changelogs_server_idx ON server_changelogs (server_id, created_at DESC);
CREATE INDEX server_changelogs_created_idx ON server_changelogs (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE server_changelogs;
-- +goose StatementEnd
