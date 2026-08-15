-- Optional seed data for the 5 recognized shop benefits (see
-- routes/shop/assets/benefits.go for what each ID actually does). Not part
-- of the schema migration in exp/shopbenefits.sql — these are just starter
-- catalog rows and can equally well be created/edited through the Arcadia
-- panel's existing Shop Items / Shop Item Benefits screens instead. Prices
-- are a starting point, not a final call — tune freely.
--
-- created_by/updated_by must be a real staff user_id; replace
-- REPLACE_WITH_STAFF_USER_ID before running.

insert into shop_item_benefits (id, name, description, target_types, created_by, updated_by) values
    ('priority_boost', 'Priority Boost', 'Bumps the bot toward the top of listing sort order while active.', array['bot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID'),
    ('featured_slot', 'Featured Homepage Slot', 'Shows the bot in the home page''s Featured section while active.', array['bot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID'),
    ('supporter_badge', 'Supporter Badge', 'A permanent cosmetic badge on the bot''s card and page. No effect on ranking or premium.', array['bot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID'),
    ('vote_blitz', 'Vote Blitz', 'Halves the bot''s per-user vote cooldown while active.', array['bot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID'),
    ('premium_days', 'Bonus Premium Days', 'Extends the bot''s premium period.', array['bot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID');

insert into shop_items (id, name, cents, target_types, benefits, created_by, updated_by, duration, description) values
    ('priority-boost-24h', 'Priority Boost (24h)', 40, array['bot'], array['priority_boost'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID', 24, 'Priority placement in bot listings for 24 hours.'),
    ('featured-slot-48h', 'Featured Homepage Slot (48h)', 150, array['bot'], array['featured_slot'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID', 48, 'A spot in the home page Featured section for 48 hours.'),
    ('supporter-badge', 'Supporter Badge', 20, array['bot'], array['supporter_badge'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID', 0, 'A permanent cosmetic Supporter badge.'),
    ('vote-blitz-24h', 'Vote Blitz (24h)', 60, array['bot'], array['vote_blitz'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID', 24, 'Halves the vote cooldown for 24 hours.'),
    ('premium-days-7', 'Bonus Premium (+7 days)', 25, array['bot'], array['premium_days'], 'REPLACE_WITH_STAFF_USER_ID', 'REPLACE_WITH_STAFF_USER_ID', 168, 'Extends premium by 7 days.');
