-- Renames four staff permissions that were named after bots even though
-- they gate bot AND server actions (Approve/Deny/Claim/etc. and
-- CertifyAdd/Remove branch on TargetType; ForceRemove also covers packs):
--
--   review_bots         -> review_entities
--   certify_bots        -> certify_entities
--   force_remove_bots   -> force_remove_entities
--   marker_bot_reviewer -> marker_reviewer
--
-- This is a straight rename of already-flat permission names (not a
-- namespace migration — see exp/rewrite/flatperms.sql for that one-time
-- job, already run). Safe to run more than once: array_replace on a name
-- that's already been renamed is a no-op.
--
-- Run it inside the transaction it opens, then COMMIT:
--   psql "$DATABASE_URL" -f exp/rewrite/rename_reviewer_perms.sql

BEGIN;

UPDATE staff_positions
   SET perms = array_replace(array_replace(array_replace(array_replace(
           perms,
           'review_bots', 'review_entities'),
           'certify_bots', 'certify_entities'),
           'force_remove_bots', 'force_remove_entities'),
           'marker_bot_reviewer', 'marker_reviewer')
 WHERE perms && ARRAY['review_bots', 'certify_bots', 'force_remove_bots', 'marker_bot_reviewer'];

UPDATE staff_members
   SET perm_overrides = array_replace(array_replace(array_replace(array_replace(
           perm_overrides,
           'review_bots', 'review_entities'),
           'certify_bots', 'certify_entities'),
           'force_remove_bots', 'force_remove_entities'),
           'marker_bot_reviewer', 'marker_reviewer')
 WHERE perm_overrides && ARRAY['review_bots', 'certify_bots', 'force_remove_bots', 'marker_bot_reviewer'];

UPDATE staff_disciplinary_types
   SET perm_limits = array_replace(array_replace(array_replace(array_replace(
           perm_limits,
           'review_bots', 'review_entities'),
           'certify_bots', 'certify_entities'),
           'force_remove_bots', 'force_remove_entities'),
           'marker_bot_reviewer', 'marker_reviewer')
 WHERE perm_limits && ARRAY['review_bots', 'certify_bots', 'force_remove_bots', 'marker_bot_reviewer'];

COMMIT;
