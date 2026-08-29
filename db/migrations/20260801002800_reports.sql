-- +goose Up
-- +goose StatementBegin
create table reports (
    id uuid primary key default gen_random_uuid(),
    target_type text not null,
    target_id text not null,
    reporter_id text not null,
    reason text not null,
    description text not null,
    status text not null default 'open' check (status in ('open', 'under_review', 'resolved', 'dismissed')),
    resolved_by text,
    resolution_note text,
    created_at timestamptz not null default now(),
    resolved_at timestamptz
);
create index reports_target_idx on reports(target_type, target_id);
create index reports_status_idx on reports(status);
create unique index reports_reporter_target_open_idx on reports(target_type, target_id, reporter_id) where status = 'open';
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/reports.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
