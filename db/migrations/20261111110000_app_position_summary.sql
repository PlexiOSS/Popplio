-- +goose Up
-- +goose StatementBegin
ALTER TABLE app_positions ADD COLUMN summary text DEFAULT '' NOT NULL;
UPDATE app_positions SET summary = 'Review bots and servers, moderate the community and help members. You need to be 16 or older.' WHERE id = 'staff';
UPDATE app_positions SET summary = 'Help build Omniplex: the website, API, SDKs and bots. Some Go or TypeScript and PostgreSQL experience needed.' WHERE id = 'dev';
UPDATE app_positions SET summary = 'Partner your bot, server or business with Omniplex and grow together.' WHERE id = 'partners';
UPDATE app_positions SET summary = 'Earn the Certified badge for an established bot: 50+ servers, 15+ unique views or 50+ votes.' WHERE id = 'certification';
UPDATE app_positions SET summary = 'Earn the Certified badge for an established server: 100+ members, 15+ unique views or 50+ votes.' WHERE id = 'server_certification';
UPDATE app_positions SET summary = 'Fixed what got your bot denied? Ask for another review.' WHERE id = 'resubmit';
UPDATE app_positions SET summary = 'Banned from Omniplex? Tell us what happened and what has changed.' WHERE id = 'banappeal';
UPDATE app_positions SET summary = 'Try new features before anyone else and earn the Beta Tester badge.' WHERE id = 'beta';
UPDATE app_positions SET summary = 'Think your bot or server was denied, removed or vote banned by mistake? Management will take another look.' WHERE id = 'listing_appeal';
UPDATE app_positions SET summary = 'Pitch your bot, server or pack to be featured in a blog post or collection.' WHERE id = 'spotlight';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE app_positions DROP COLUMN summary;
-- +goose StatementEnd
