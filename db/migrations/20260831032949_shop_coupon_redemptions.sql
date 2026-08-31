-- +goose Up
-- +goose StatementBegin
-- shop_coupons.id has never had a primary key or unique constraint --
-- every other table in this schema has one, this looks like a genuine
-- oversight rather than a deliberate choice. Table is empty in prod as of
-- this migration, so adding it now is a no-op on existing data and it's
-- needed regardless for the FK below.
ALTER TABLE public.shop_coupons ADD CONSTRAINT shop_coupons_pkey PRIMARY KEY (id);

-- Tracks each time a shop coupon is redeemed, so max_uses (a global cap)
-- and reuse_wait_duration (a per-target cooldown -- an entity can't reuse
-- the same coupon again until this many hours have passed since its own
-- last redemption of it) can actually be enforced. Neither shop_coupons
-- column had anywhere to read usage history from before this.
CREATE TABLE public.shop_coupon_redemptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL PRIMARY KEY,
    coupon_id text NOT NULL REFERENCES public.shop_coupons(id) ON DELETE CASCADE,
    target_type text NOT NULL,
    target_id text NOT NULL,
    redeemed_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX shop_coupon_redemptions_coupon_id_idx ON public.shop_coupon_redemptions(coupon_id);
CREATE INDEX shop_coupon_redemptions_target_idx ON public.shop_coupon_redemptions(target_type, target_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.shop_coupon_redemptions;
ALTER TABLE public.shop_coupons DROP CONSTRAINT IF EXISTS shop_coupons_pkey;
-- +goose StatementEnd
