-- +goose Up
-- +goose StatementBegin
ALTER TABLE auto_vote_subscriptions ADD COLUMN card_fingerprint text;
CREATE INDEX auto_vote_subscriptions_card_idx ON auto_vote_subscriptions (card_fingerprint) WHERE card_fingerprint IS NOT NULL;
CREATE INDEX entity_votes_auto_created_idx ON entity_votes (created_at) WHERE upvote AND NOT void;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS entity_votes_auto_created_idx;
DROP INDEX IF EXISTS auto_vote_subscriptions_card_idx;
ALTER TABLE auto_vote_subscriptions DROP COLUMN card_fingerprint;
-- +goose StatementEnd
