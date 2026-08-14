-- ADD COLUMN with a constant DEFAULT is metadata-only on Postgres 11+ (no
-- table rewrite), safe and instant regardless of table size.
alter table packs add column pack_type text not null default 'bot';

-- Split into NOT VALID + a separate VALIDATE CONSTRAINT rather than a
-- single ADD CONSTRAINT ... CHECK: the single-statement form takes an
-- ACCESS EXCLUSIVE lock (blocks reads AND writes on packs) for as long as
-- it takes to scan and validate every existing row. NOT VALID applies the
-- constraint to new/future writes immediately without scanning anything;
-- VALIDATE CONSTRAINT then does the existing-row scan under a much
-- lighter SHARE UPDATE EXCLUSIVE lock, which still allows concurrent
-- reads and writes to packs while it runs.
alter table packs add constraint packs_pack_type_check check (pack_type in ('bot', 'server', 'emoji')) not valid;
alter table packs validate constraint packs_pack_type_check;
