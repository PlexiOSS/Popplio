-- Per-user, per-category notification preferences, plus a category tag on
-- alerts themselves so the preference actually means something.
--
-- No row for a (user, category) pair means enabled -- this is an opt-out
-- model, so existing users need no backfill and keep seeing everything
-- until they explicitly mute a category. Categories are intentionally
-- topic-level (see types.AlertCategory in Go), not one row per event type.
--
-- The `alerts` table itself predates migration tracking in this repo (see
-- types.Alert's ITag doc comment) and isn't defined anywhere else in
-- exp/ -- this only adds the one column it's missing.

alter table alerts add column if not exists category text not null default 'general';

create table if not exists user_notification_prefs (
    user_id  text not null,
    category text not null,
    enabled  boolean not null default true,
    primary key (user_id, category)
);
