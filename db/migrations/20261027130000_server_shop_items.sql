-- +goose Up
-- +goose StatementBegin
UPDATE shop_item_benefits
SET target_types = ARRAY['bot', 'server']
WHERE id IN ('featured_slot', 'supporter_badge', 'vote_blitz')
  AND NOT ('server' = ANY(target_types));
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shop_item_benefits SET description = 'Shows the bot or server in the home page''s Featured section while active.' WHERE id = 'featured_slot';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE shop_item_benefits SET description = 'A permanent cosmetic badge on the listing''s card and page. No effect on ranking or premium.' WHERE id = 'supporter_badge';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE shop_item_benefits SET description = 'Halves the listing''s per-user vote cooldown while active.' WHERE id = 'vote_blitz';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE shop_item_benefits SET description = 'Bumps the listing toward the top of listing sort order while active.' WHERE id = 'priority_boost';
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE shop_item_benefits SET description = 'Extends the listing''s premium period.' WHERE id = 'premium_days';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shop_items
SET target_types = ARRAY['bot', 'server']
WHERE id IN ('featured-slot-48h', 'premium-days-7', 'priority-boost-24h', 'supporter-badge', 'vote-blitz-24h')
  AND NOT ('server' = ANY(target_types));
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE shop_items SET description = 'Priority placement in listings for 24 hours.' WHERE id = 'priority-boost-24h';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE shop_item_benefits SET target_types = ARRAY['bot'] WHERE id IN ('featured_slot', 'supporter_badge', 'vote_blitz');
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE shop_items SET target_types = ARRAY['bot'] WHERE id IN ('featured-slot-48h', 'premium-days-7', 'priority-boost-24h', 'supporter-badge', 'vote-blitz-24h');
-- +goose StatementEnd
