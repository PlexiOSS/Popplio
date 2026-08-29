-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_votes (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    target_id TEXT NOT NULL,
    target_type TEXT NOT NULL,
    author TEXT NOT NULL REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE CASCADE,
    upvote BOOLEAN NOT NULL,
    void BOOLEAN NOT NULL DEFAULT FALSE,
    void_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- POST-IMPL note from the original exp/votesv2.sql, never turned into an
-- actual statement here: remove pack_votes and votes tables (their
-- replacement, entity_votes, is what this file creates).
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/votesv2.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
