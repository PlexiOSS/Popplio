-- +goose Up
-- +goose StatementBegin
ALTER TABLE entity_page_styles ADD COLUMN unlocked_until timestamp with time zone;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO shop_item_benefits (id, name, description, created_by, updated_by, target_types)
VALUES ('page_style', 'Page Style', 'Use a custom background, gradient and accent colour on the listing''s page without premium.', '510065483693817867', '510065483693817867', '{bot,server}')
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO shop_items (id, name, cents, target_types, benefits, created_by, updated_by, duration, description)
VALUES ('page-style-30d', 'Page Style (30 days)', 300, '{bot,server}', '{page_style}', '510065483693817867', '510065483693817867', 720, 'A custom background, gradient and accent colour on the listing''s page for 30 days, without premium. Buying again adds another 30 days.')
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM shop_items WHERE id = 'page-style-30d' AND NOT EXISTS (SELECT 1 FROM shop_purchases WHERE item_id = 'page-style-30d');
DELETE FROM shop_item_benefits WHERE id = 'page_style';
ALTER TABLE entity_page_styles DROP COLUMN unlocked_until;
-- +goose StatementEnd
