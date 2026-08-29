-- +goose Up
-- +goose StatementBegin
alter table webhooks add column if not exists hmac_auth boolean not null default false;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/webhookhmacauth.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
