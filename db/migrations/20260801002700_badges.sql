-- +goose Up
-- +goose StatementBegin
-- A generic, staff-managed badge system: a catalog of purely decorative
-- badges plus a flexible assignment table, so adding a new one is a new
-- catalog row and an assignment — never a new column, a new backend flag,
-- or a new frontend branch.
--
-- Deliberately separate from the *functional* badges already baked into
-- bots/servers/users (premium, certified, supporter_badge, developer,
-- bug_hunters, etc.) — those gate real behaviour or are synced from a
-- Discord role automatically, and stay exactly as they are. This table is
-- for the "OG Member" / "Beta Tester" / "Contest Winner" kind of badge,
-- assigned by a staff member through the same Actions menu as every other
-- staff action.

create table if not exists badges (
    id text primary key,
    name text not null,
    description text not null default '',
    icon text not null,
    color text not null default 'default'
        check (color in ('default', 'success', 'warning', 'danger', 'info', 'premium')),
    target_types text[] not null default '{}',
    created_at timestamptz not null default now(),
    created_by text not null,
    last_updated timestamptz not null default now(),
    updated_by text not null
);

create table if not exists entity_badges (
    itag uuid primary key default uuid_generate_v4(),
    target_type text not null,
    target_id text not null,
    badge_id text not null references badges(id) on delete cascade,
    reason text not null default '',
    awarded_by text not null,
    created_at timestamptz not null default now(),
    unique (target_type, target_id, badge_id)
);

create index if not exists entity_badges_target_idx on entity_badges (target_type, target_id);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/badges.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
