-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
create index concurrently if not exists entity_votes_target_created_idx on entity_votes(target_type, target_id, created_at) where void = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index concurrently if exists entity_votes_target_created_idx;
-- +goose StatementEnd
