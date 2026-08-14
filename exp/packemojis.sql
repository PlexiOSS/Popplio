create table pack_emojis (
    id uuid primary key default gen_random_uuid(),
    pack_url text not null references packs(url) on delete cascade,
    name text not null,
    animated boolean not null default false,
    position int not null default 0,
    created_at timestamptz not null default now()
);
create index pack_emojis_pack_url_idx on pack_emojis(pack_url);
