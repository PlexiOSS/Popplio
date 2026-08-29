-- +goose Up
-- +goose StatementBegin
-- Reordered from the original exp/partnersv2.sql, which created `partners`
-- (with a FK to partner_types) before `partner_types` itself existed --
-- invisible on prod for the same reason as the vanityv2/votesv2 comment
-- bugs: no ON_ERROR_STOP/transaction wrapping, so it likely errored out on
-- prod too and was silently re-run in the right order by hand. Table
-- creation order swapped here so it just works; no content changes.
CREATE TABLE partner_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    short TEXT NOT NULL,
    icon TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE partners (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    short TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(user_id),
    image TEXT NOT NULL,
    links jsonb NOT NULL,
    type TEXT NOT NULL REFERENCES partner_types(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'bot',
    'Bot Partner',
    'Meet some great discord bots officially partnered with us!',
    'bxs:bot'
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'featured',
    'Featured Partners',
    'These are the partners that we have chosen to featured on our website.',
    'material-symbols:featured-play-list'
);

INSERT INTO partner_types (
    id,
    name,
    short,
    icon
) VALUES (
    'botlist',
    'Bot List Partner',
    'These are the bot lists who we trust!',
    'material-symbols:list'
);
-- +goose StatementEnd

-- +goose Down
-- One-way historical port from exp/partnersv2.sql -- no rollback written;
-- this reflects a change already long-applied to prod, not new work.
SELECT 1;
