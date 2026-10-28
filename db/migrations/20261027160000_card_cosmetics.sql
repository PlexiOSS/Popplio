-- +goose Up
-- +goose StatementBegin
ALTER TABLE bots
    ADD COLUMN highlighted_until timestamp with time zone,
    ADD COLUMN card_theme boolean DEFAULT false NOT NULL,
    ADD COLUMN card_color text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE servers
    ADD COLUMN highlighted_until timestamp with time zone,
    ADD COLUMN card_theme boolean DEFAULT false NOT NULL,
    ADD COLUMN card_color text;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO shop_item_benefits (id, name, description, target_types, created_by, updated_by) VALUES
    ('highlight', 'Highlighted Card', 'Gives the listing''s card an accent glow in browse, search and home page grids while active.', ARRAY['bot', 'server'], 'system', 'system'),
    ('spotlight_slot', 'Spotlight Slot', 'Shows the listing in the home page''s Spotlight section while active.', ARRAY['bot', 'server'], 'system', 'system'),
    ('card_theme', 'Card Theme', 'Lets the owner tint the listing''s card and page header with a colour of their choice. Permanent.', ARRAY['bot', 'server'], 'system', 'system')
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO shop_items (id, name, cents, target_types, benefits, created_by, updated_by, duration, description) VALUES
    ('highlight-7d', 'Highlighted Card (7 days)', 80, ARRAY['bot', 'server'], ARRAY['highlight'], 'system', 'system', 168, 'An accent glow on the listing''s card everywhere it appears, for 7 days.'),
    ('spotlight-24h', 'Spotlight (24h)', 200, ARRAY['bot', 'server'], ARRAY['spotlight_slot'], 'system', 'system', 24, 'A place in the home page Spotlight section for 24 hours.'),
    ('card-theme', 'Card Theme', 120, ARRAY['bot', 'server'], ARRAY['card_theme'], 'system', 'system', 0, 'Pick a colour to tint the listing''s card and page header. Permanent.'),
    ('launch-bundle-48h', 'Launch Bundle (48h)', 260, ARRAY['bot', 'server'], ARRAY['featured_slot', 'priority_boost', 'vote_blitz'], 'system', 'system', 48, 'Featured slot, Priority Boost and Vote Blitz together for 48 hours, for less than buying them separately.')
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM shop_items WHERE id IN ('highlight-7d', 'spotlight-24h', 'card-theme', 'launch-bundle-48h')
    AND NOT EXISTS (SELECT 1 FROM shop_purchases p WHERE p.item_id = shop_items.id)
    AND NOT EXISTS (SELECT 1 FROM shop_holds h WHERE h.item = shop_items.id);
-- +goose StatementEnd
-- +goose StatementBegin
DELETE FROM shop_item_benefits WHERE id IN ('highlight', 'spotlight_slot', 'card_theme');
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE servers DROP COLUMN highlighted_until, DROP COLUMN card_theme, DROP COLUMN card_color;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE bots DROP COLUMN highlighted_until, DROP COLUMN card_theme, DROP COLUMN card_color;
-- +goose StatementEnd
