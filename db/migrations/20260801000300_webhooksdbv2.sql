-- +goose Up
-- +goose StatementBegin
-- Columns event_whitelist, name, failed_requests, and the UNIQUE(id)
-- constraint (named id_unique) were added on the port: none of them are
-- captured by any exp/*.sql file, yet prod's live table has all of them
-- -- confirmed via \d webhooks against prod. Added ad hoc, outside any
-- tracked script, same as the vanity.itag PK and the citext/uuid-ossp/
-- pg_trgm extensions. The id unique constraint specifically is required:
-- webhook_logs.webhook_id references webhooks(id), and a FK target needs
-- a unique/PK constraint to reference.
CREATE TABLE webhooks (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    url TEXT NOT NULL CHECK (url <> ''),
    secret TEXT NOT NULL CHECK (secret <> ''),
    broken BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook is broken
    simple_auth BOOLEAN NOT NULL DEFAULT FALSE, -- Whether or not the webhook should use simple auth
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    event_whitelist TEXT[] NOT NULL DEFAULT '{}',
    name TEXT NOT NULL DEFAULT 'My untitled webhook',
    failed_requests INTEGER NOT NULL DEFAULT 0,
    UNIQUE (target_id, target_type),
    CONSTRAINT id_unique UNIQUE (id)
);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/webhooksdbv2.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
