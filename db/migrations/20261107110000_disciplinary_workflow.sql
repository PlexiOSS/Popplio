-- +goose Up
-- +goose StatementBegin
ALTER TABLE staff_disciplinary ADD COLUMN issued_by text;
ALTER TABLE staff_disciplinary ADD COLUMN starts_at timestamp with time zone;
ALTER TABLE staff_disciplinary ADD COLUMN decided_by text;
ALTER TABLE staff_disciplinary ADD COLUMN decided_at timestamp with time zone;
ALTER TABLE staff_disciplinary ADD COLUMN decision_note text;
ALTER TABLE staff_disciplinary ADD COLUMN revoked_by text;
ALTER TABLE staff_disciplinary ADD COLUMN revoked_at timestamp with time zone;
ALTER TABLE staff_disciplinary ADD COLUMN revoke_reason text;
UPDATE staff_disciplinary SET starts_at = created_at WHERE starts_at IS NULL;
CREATE INDEX staff_disciplinary_user_id_idx ON staff_disciplinary (user_id);
CREATE INDEX staff_disciplinary_state_idx ON staff_disciplinary (state);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX staff_disciplinary_state_idx;
DROP INDEX staff_disciplinary_user_id_idx;
ALTER TABLE staff_disciplinary DROP COLUMN revoke_reason;
ALTER TABLE staff_disciplinary DROP COLUMN revoked_at;
ALTER TABLE staff_disciplinary DROP COLUMN revoked_by;
ALTER TABLE staff_disciplinary DROP COLUMN decision_note;
ALTER TABLE staff_disciplinary DROP COLUMN decided_at;
ALTER TABLE staff_disciplinary DROP COLUMN decided_by;
ALTER TABLE staff_disciplinary DROP COLUMN starts_at;
ALTER TABLE staff_disciplinary DROP COLUMN issued_by;
-- +goose StatementEnd
