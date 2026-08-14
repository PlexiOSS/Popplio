alter table packs add column pack_type text not null default 'bot';
alter table packs add constraint packs_pack_type_check check (pack_type in ('bot', 'server', 'emoji'));
