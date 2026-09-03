-- +goose Up
-- +goose StatementBegin
-- InsertAutoReport's `ON CONFLICT DO NOTHING` has always been a no-op --
-- there was no unique constraint for it to land on (the PK is a random
-- UUID, always unique), so every periodic re-scan that still flagged a
-- bot/server filed a brand-new duplicate report, even for something staff
-- had already reviewed and dismissed. Clean up existing exact duplicates
-- first (keep the oldest row per group, which is the one staff already
-- triaged in every case surveyed) so the constraint below can be added.
DELETE FROM reports r
USING (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY target_type, target_id, reporter_id, description
               ORDER BY created_at ASC
           ) AS rn
    FROM reports
    WHERE reporter_id = 'system:moderation'
) dup
WHERE r.id = dup.id AND dup.rn > 1;
-- +goose StatementEnd

-- +goose StatementBegin
-- Scoped to system:moderation only -- real user reports should never be
-- deduped against each other just for sharing description text.
CREATE UNIQUE INDEX reports_auto_dedup_idx ON reports (target_type, target_id, reporter_id, description) WHERE reporter_id = 'system:moderation';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX reports_auto_dedup_idx;
-- +goose StatementEnd
