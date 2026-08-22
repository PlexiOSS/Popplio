-- Staff review templates were bot-only in practice (GET /list/staff-templates
-- itself only ever documented "used for reviewing bots"), with no column to
-- say so. Existing rows default to 'bot' so nothing already in the table
-- silently becomes ambiguous; new server-specific templates get 'server'.
alter table staff_templates add column entity_type text not null default 'bot';
