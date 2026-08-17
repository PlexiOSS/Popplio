-- Extends servers with the same shop-benefit columns bots got in
-- exp/shopbenefits.sql, now that servers support premium/shop/certification
-- the same way bots do. All four ADD COLUMNs are metadata-only on
-- Postgres 11+ (constant/NULL default, no table rewrite), safe and
-- instant regardless of table size.
alter table servers add column boosted_until timestamptz;
alter table servers add column featured_until timestamptz;
alter table servers add column supporter_badge boolean not null default false;
alter table servers add column vote_blitz_until timestamptz;
