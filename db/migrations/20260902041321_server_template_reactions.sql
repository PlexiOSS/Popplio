-- Like/dislike reactions on server templates. Deliberately not modeled on
-- entity_votes -- votes there mean "cast periodically, cooldown-gated,
-- convertible to shop credits," which isn't what this is. A reaction is a
-- single persistent choice per (template, user): liking then disliking
-- switches it, clicking the active one again clears it. One row per
-- (template_id, user_id) is both the storage and the enforcement -- no
-- separate "has this user already reacted" check needed.
--
-- Counts are computed live (COUNT ... FILTER) rather than cached on
-- server_templates, on purpose: a cached count that drifts from the real
-- rows is exactly the class of bug the vote-count incident on
-- 2026-09-01 (see server status page) was. Template reaction volume is
-- small enough that a live COUNT costs nothing worth caching for.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE server_template_reactions (
    template_id uuid NOT NULL REFERENCES server_templates(id) ON DELETE CASCADE,
    user_id text NOT NULL,
    liked boolean NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    PRIMARY KEY (template_id, user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE server_template_reactions;
-- +goose StatementEnd
