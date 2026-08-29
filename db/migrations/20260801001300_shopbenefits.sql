-- +goose Up
-- +goose StatementBegin
-- Backs the new shop-purchase benefits (priority boost, featured homepage
-- slot, cosmetic supporter badge, vote blitz). All four ADD COLUMNs are
-- metadata-only on Postgres 11+ (constant/NULL default, no table rewrite),
-- safe and instant regardless of table size.
-- IF NOT EXISTS added on the port: genesis_foundational_tables.sql already
-- captures these columns on bots (it's a live dump of prod, which already
-- has this change applied), so on a fresh database this becomes a no-op.
alter table bots add column if not exists boosted_until timestamptz;
alter table bots add column if not exists featured_until timestamptz;
alter table bots add column if not exists supporter_badge boolean not null default false;
alter table bots add column if not exists vote_blitz_until timestamptz;

-- Purchase history / ledger for shop_items bought with entity vote credits.
-- A fresh table, so CREATE TABLE IF NOT EXISTS takes no lock on anything
-- that already has data in it.
create table if not exists shop_purchases (
    id uuid primary key default uuid_generate_v4(),
    target_type text not null,
    target_id text not null,
    item_id text not null references shop_items(id),
    cents double precision not null,
    created_at timestamptz not null default now()
);

create index if not exists shop_purchases_target_idx on shop_purchases (target_type, target_id);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/shopbenefits.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
