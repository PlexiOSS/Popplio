-- +goose Up
-- +goose StatementBegin
-- Staff review templates were bot-only in practice (GET /list/staff-templates
-- itself only ever documented "used for reviewing bots"), with no column to
-- say so. Existing rows default to 'bot' so nothing already in the table
-- silently becomes ambiguous; new server-specific templates get 'server'.
alter table staff_templates add column if not exists entity_type text not null default 'bot';
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/stafftemplateentitytype.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
