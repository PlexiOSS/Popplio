alter table bots add column if not exists spotlighted_until timestamptz;
alter table servers add column if not exists spotlighted_until timestamptz;
