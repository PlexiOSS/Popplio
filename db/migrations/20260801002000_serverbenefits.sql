-- +goose Up
-- +goose StatementBegin
-- Extends servers with the same shop-benefit columns bots got in
-- exp/shopbenefits.sql, now that servers support premium/shop/certification
-- the same way bots do. All four ADD COLUMNs are metadata-only on
-- Postgres 11+ (constant/NULL default, no table rewrite), safe and
-- instant regardless of table size.
alter table servers add column if not exists boosted_until timestamptz;
alter table servers add column if not exists featured_until timestamptz;
alter table servers add column if not exists supporter_badge boolean not null default false;
alter table servers add column if not exists vote_blitz_until timestamptz;
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/serverbenefits.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
