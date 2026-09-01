-- Data repair for the vote-credit redemption bug: RedeemVotesByItags and
-- RedeemAllVotesForTarget used to set void = true on every vote they
-- redeemed, reusing the same flag as an automated/staff vote reset. Since
-- every public vote-count query filters on void = false, this silently
-- deflated a bot/server/team's public vote total (and approximate_votes)
-- every time its owner cashed votes in for shop credits. The application
-- code no longer sets void = true on redemption (see db/queries/votes.sql);
-- this migration un-voids the rows that were already wrongly voided this
-- way, and recomputes the three cached approximate_votes columns from the
-- corrected data. Packs aren't included -- their vote count is computed
-- live from entity_votes, never cached.

-- +goose Up
-- +goose StatementBegin
UPDATE entity_votes
SET void = false
WHERE void = true AND void_reason = 'Vote credits redeemed';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE bots t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = 'bot' AND void = false GROUP BY target_id) v
WHERE t.bot_id = v.target_id;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE servers t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = 'server' AND void = false GROUP BY target_id) v
WHERE t.server_id = v.target_id;
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE teams t SET approximate_votes = v.count FROM
    (SELECT target_id, COUNT(*) FILTER (WHERE upvote) - COUNT(*) FILTER (WHERE NOT upvote) AS count
     FROM entity_votes WHERE target_type = 'team' AND void = false GROUP BY target_id) v
WHERE t.id::text = v.target_id;
-- +goose StatementEnd

-- This is a data repair, not a schema change -- there's nothing meaningful
-- to roll back to (re-voiding the rows this fixed would just reintroduce
-- the bug for anyone who redeems credits after the rollback). Down is
-- intentionally a no-op.
-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
