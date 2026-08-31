-- name: GetShopPurchasesByTarget :many
SELECT id, target_type, target_id, item_id, cents, created_at
FROM shop_purchases
WHERE target_id = $1 AND target_type = $2
ORDER BY created_at DESC;

-- ApplyBenefit (popplio/routes/shop/assets/benefits.go) dispatches on a
-- fixed, never-user-controlled (targetType, benefitID) pair -- one explicit
-- query per combination, same reasoning as db/queries/payments.sql.

-- name: ApplyBotPremiumDays :exec
UPDATE bots SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE bot_id = $2;

-- name: ApplyServerPremiumDays :exec
UPDATE servers SET start_premium_period = NOW(), premium_period_length = make_interval(hours => $1), premium = true WHERE server_id = $2;

-- name: ApplyBotPriorityBoost :exec
UPDATE bots SET boosted_until = GREATEST(COALESCE(boosted_until, NOW()), NOW()) + make_interval(hours => $1) WHERE bot_id = $2;

-- name: ApplyServerPriorityBoost :exec
UPDATE servers SET boosted_until = GREATEST(COALESCE(boosted_until, NOW()), NOW()) + make_interval(hours => $1) WHERE server_id = $2;

-- name: ApplyBotFeaturedSlot :exec
UPDATE bots SET featured_until = GREATEST(COALESCE(featured_until, NOW()), NOW()) + make_interval(hours => $1) WHERE bot_id = $2;

-- name: ApplyServerFeaturedSlot :exec
UPDATE servers SET featured_until = GREATEST(COALESCE(featured_until, NOW()), NOW()) + make_interval(hours => $1) WHERE server_id = $2;

-- name: ApplyBotSupporterBadge :exec
UPDATE bots SET supporter_badge = true WHERE bot_id = $1;

-- name: ApplyServerSupporterBadge :exec
UPDATE servers SET supporter_badge = true WHERE server_id = $1;

-- name: ApplyBotVoteBlitz :exec
UPDATE bots SET vote_blitz_until = GREATEST(COALESCE(vote_blitz_until, NOW()), NOW()) + make_interval(hours => $1) WHERE bot_id = $2;

-- name: ApplyServerVoteBlitz :exec
UPDATE servers SET vote_blitz_until = GREATEST(COALESCE(vote_blitz_until, NOW()), NOW()) + make_interval(hours => $1) WHERE server_id = $2;

-- name: GetShopItems :many
SELECT id, name, cents, target_types, benefits, created_at, last_updated, created_by, updated_by, duration, description
FROM shop_items
ORDER BY created_at DESC;

-- name: GetShopItemByID :one
SELECT id, name, cents, target_types, benefits, created_at, last_updated, created_by, updated_by, duration, description
FROM shop_items
WHERE id = $1;

-- name: GetShopItemBenefits :many
SELECT id, name, description, created_at, last_updated, created_by, updated_by, target_types
FROM shop_item_benefits
ORDER BY created_at DESC;

-- name: GetPublicShopCoupons :many
-- Only coupons someone could actually redeem right now -- a disabled or
-- expired coupon showing up in a public "available offers" list would be
-- actively misleading, not just harmlessly stale.
SELECT id, code, public, max_uses, created_at, created_by, last_updated, updated_by, reuse_wait_duration, expiry, applicable_items, requirements, allowed_users, usable, target_types, cents
FROM shop_coupons
WHERE public = true
AND usable = true
AND (expiry IS NULL OR created_at + make_interval(hours => expiry) > NOW())
AND (max_uses IS NULL OR (SELECT COUNT(*) FROM shop_coupon_redemptions WHERE coupon_id = shop_coupons.id) < max_uses)
ORDER BY created_at DESC;

-- name: GetRedeemableCreditBatches :many
SELECT id, credits, redeemed_credits FROM entity_vote_redeem_logs WHERE target_id = $1 AND target_type = $2 AND redeemed_credits < credits ORDER BY created_at ASC FOR UPDATE;

-- name: SpendCreditBatch :exec
UPDATE entity_vote_redeem_logs SET redeemed_credits = redeemed_credits + $1, redeemed_at = NOW() WHERE id = $2;

-- name: InsertShopPurchase :exec
INSERT INTO shop_purchases (target_type, target_id, item_id, cents) VALUES ($1, $2, $3, $4);

-- Panel shop-item CRUD (db/queries/shop.sql's GetShopItems/GetShopItemByID
-- are reused for the panel's own list/existence checks where the columns match).

-- name: CountShopItemByID :one
SELECT EXISTS(SELECT 1 FROM shop_items WHERE id = $1);

-- name: InsertShopItem :exec
INSERT INTO shop_items (id, name, cents, target_types, benefits, created_by, updated_by, duration, description)
VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8);

-- name: UpdateShopItem :exec
UPDATE shop_items SET name = $1, cents = $2, target_types = $3, benefits = $4, last_updated = NOW(), updated_by = $5, duration = $6, description = $7 WHERE id = $8;

-- name: DeleteShopItem :exec
DELETE FROM shop_items WHERE id = $1;

-- name: CountShopItemBenefitByID :one
SELECT EXISTS(SELECT 1 FROM shop_item_benefits WHERE id = $1);

-- name: InsertShopItemBenefit :exec
INSERT INTO shop_item_benefits (id, name, description, target_types, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateShopItemBenefit :exec
UPDATE shop_item_benefits SET name = $1, description = $2, last_updated = NOW(), updated_by = $3, target_types = $4 WHERE id = $5;

-- name: CountShopItemsUsingBenefit :one
SELECT EXISTS(SELECT 1 FROM shop_items WHERE sqlc.arg('id')::text = ANY(benefits));

-- name: DeleteShopItemBenefit :exec
DELETE FROM shop_item_benefits WHERE id = $1;

-- name: ListShopCoupons :many
SELECT id, code, public, max_uses, created_at, created_by, last_updated, updated_by, reuse_wait_duration, expiry, applicable_items, cents, requirements, allowed_users, usable, target_types
FROM shop_coupons ORDER BY created_at DESC;

-- name: InsertShopCoupon :exec
INSERT INTO shop_coupons (id, code, public, max_uses, created_by, updated_by, reuse_wait_duration, expiry, applicable_items, cents, requirements, allowed_users, usable, target_types)
VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: CountShopCouponByID :one
SELECT EXISTS(SELECT 1 FROM shop_coupons WHERE id = $1);

-- name: UpdateShopCoupon :exec
UPDATE shop_coupons SET code = $1, public = $2, max_uses = $3, reuse_wait_duration = $4, expiry = $5, applicable_items = $6, cents = $7, requirements = $8, updated_by = $9, last_updated = NOW(), allowed_users = $10, usable = $11, target_types = $12 WHERE id = $13;

-- name: DeleteShopCoupon :exec
DELETE FROM shop_coupons WHERE id = $1;

-- Coupon redemption (purchase_shop_item route) -----------------------------

-- name: GetShopCouponByCode :one
SELECT id, code, public, max_uses, created_at, created_by, last_updated, updated_by, reuse_wait_duration, expiry, applicable_items, requirements, allowed_users, usable, target_types, cents
FROM shop_coupons
WHERE code = $1;

-- name: CountShopCouponRedemptions :one
SELECT COUNT(*) FROM shop_coupon_redemptions WHERE coupon_id = $1;

-- name: GetLastShopCouponRedemptionForTarget :one
SELECT created_at FROM shop_coupon_redemptions
WHERE coupon_id = $1 AND target_type = $2 AND target_id = $3
ORDER BY created_at DESC LIMIT 1;

-- name: InsertShopCouponRedemption :exec
INSERT INTO shop_coupon_redemptions (coupon_id, target_type, target_id, redeemed_by)
VALUES ($1, $2, $3, $4);

-- name: ListVoteCreditTiers :many
SELECT id, target_type, position, cents, votes, created_at FROM vote_credit_tiers ORDER BY position ASC;

-- name: InsertVoteCreditTier :exec
INSERT INTO vote_credit_tiers (id, target_type, position, cents, votes) VALUES ($1, $2, $3, $4, $5);

-- name: DedupTierPositions :exec
UPDATE vote_credit_tiers SET position = position + 1 WHERE position >= $1 AND id != $2;

-- name: CountVoteCreditTierByID :one
SELECT EXISTS(SELECT 1 FROM vote_credit_tiers WHERE id = $1);

-- name: UpdateVoteCreditTier :exec
UPDATE vote_credit_tiers SET position = $1, target_type = $2, cents = $3, votes = $4 WHERE id = $5;

-- name: DeleteVoteCreditTier :exec
DELETE FROM vote_credit_tiers WHERE id = $1;

-- name: ListBotWhitelist :many
SELECT bot_id, user_id, reason, created_at FROM bot_whitelist ORDER BY created_at DESC;

-- name: InsertBotWhitelist :exec
INSERT INTO bot_whitelist (user_id, bot_id, reason) VALUES ($1, $2, $3);

-- name: CountBotWhitelistByBotID :one
SELECT EXISTS(SELECT 1 FROM bot_whitelist WHERE bot_id = $1);

-- name: UpdateBotWhitelistReason :exec
UPDATE bot_whitelist SET reason = $1 WHERE bot_id = $2;

-- name: DeleteBotWhitelist :exec
DELETE FROM bot_whitelist WHERE bot_id = $1;
