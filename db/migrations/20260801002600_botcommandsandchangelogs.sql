-- +goose Up
-- +goose StatementBegin
-- Lets a bot's owner/team document its commands and post changelog
-- entries, shown as new tabs on the bot's public page. Both are scoped to
-- bots specifically (not servers/packs) per what was asked for.
--
-- bot_commands is managed as a full-list replace (PUT), same convention as
-- bots.extra_links already uses, since it's a small, freely-reorderable
-- list, not a history.
create table if not exists bot_commands (
    id uuid primary key default uuid_generate_v4(),
    bot_id text not null references bots(bot_id) on delete cascade,
    name text not null,
    description text not null default '',
    usage text not null default '',
    category text not null default '',
    position int not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists bot_commands_bot_idx on bot_commands (bot_id, position);

-- bot_changelogs is append/delete (POST/DELETE), same convention as
-- reviews, since it's a history an owner adds to over time rather than a
-- list they re-save wholesale.
create table if not exists bot_changelogs (
    id uuid primary key default uuid_generate_v4(),
    bot_id text not null references bots(bot_id) on delete cascade,
    title text not null,
    content text not null default '',
    version text not null default '',
    created_by text not null,
    created_at timestamptz not null default now()
);

create index if not exists bot_changelogs_bot_idx on bot_changelogs (bot_id, created_at desc);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/botcommandsandchangelogs.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
