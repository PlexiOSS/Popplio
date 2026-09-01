# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Redeeming vote credits was silently deflating an entity's public vote
  count. `RedeemVotesByItags`/`RedeemAllVotesForTarget` marked a redeemed
  vote `void`, the same flag every vote-count query (public totals,
  `approximate_votes`, leaderboards) filters out -- so a bot, server, or
  team's real vote count dropped every time its owner cashed votes in for
  shop credits, unrelated to any actual vote reset. Fixed at the source: a
  redeemed vote is now tracked purely via `credit_redeem` (which still
  correctly stops it from being redeemed a second time -- a new
  `EntityGetRedeemableVoteCount`/`CountRedeemableEntityVotes` excludes
  already-redeemed votes from a *new* redemption's math without hiding
  them from the entity's public count). A data-repair migration
  (`20260901025817_fix_vote_credit_redeem_incorrectly_voided_votes.sql`)
  un-voided the votes this had already wrongly voided in prod (11 rows
  across 2 bots, confirmed via a read-only count before applying) and
  recomputed the affected `approximate_votes` columns.
- Looking up a bot for submission (`GET /bots/{client_id}/meta`) crashed
  the Add Bot confirmation screen with `TypeError: null is not an object
  (evaluating 'T.flags.map')` whenever the lookup succeeded via Discord's
  own RPC endpoint, the now-preferred path since 1.7.0. That endpoint
  doesn't return flags or suggested tags/description (only the JAPI
  fallback does), and the RPC-success branch of `CheckBot` left both
  `Flags` and `Tags` unset -- a nil Go slice marshals to JSON `null`, not
  `[]`, and the frontend called `.map()` on `flags` directly. Since RPC
  succeeds for effectively every public bot, this was hitting most new
  bot submissions, not an edge case. Both fields are now explicitly
  defaulted to an empty slice on the RPC path. Audited for the same
  pattern elsewhere in the backend (any struct built across multiple
  fallback branches with an array field) -- `CheckBot` is the only place
  this shape exists.

## [1.7.2] - 2026-08-31

### Added

- A new "Server Templates" route group for user-submitted Discord server
  template listings (`discord.com/template/<code>`): `PUT
  /users/{id}/server-templates` (create), `GET /server-templates/@all`
  (paginated, filterable by `tags`/`owner`), `GET /server-templates/{id}`,
  `DELETE /users/{id}/server-templates/{id}` (owner-only). New
  `server_templates` table. Same trust model as packs -- no staff review
  queue, owner creates/deletes directly. A submission's `name` is pulled
  from Discord's own public, unauthenticated `GET
  /guilds/templates/{code}` at creation time (confirmed no bot token or
  auth needed), which is also what actually validates the code -- Popplio
  itself does no format-checking beyond length bounds.
- `GET /bots/{id}/similar` and `GET /servers/{id}/similar` -- other
  approved/certified (and, for servers, publicly listed) entries sharing
  at least one tag, ranked by how many tags they share, votes as the
  tiebreak. New `GetSimilarBots`/`GetSimilarServers` queries (tag-overlap
  via `unnest`/`INTERSECT`, no ML/embeddings -- tags are the only
  structured signal these already carry, and it's cheap, deterministic,
  and explains itself). No matches just means an empty list, not an
  error.
- `GET /votes/leaderboard` -- the most active voters, all-time, by total
  upvotes cast across every entity. Public, unauthenticated,
  `?limit=` (default 10, capped at 50). New `GetTopVoters` query,
  deliberately not scoped to the current post-reset voting cycle (counts
  every upvote ever cast, void or not) -- same lifetime-cumulative shape
  as the existing `GetTopReviewers`/Top Reviewers Discord-role sync,
  chosen so the board isn't wiped every time the monthly automated reset
  runs.
- Vote credit tiers (the votes -> credits ladder) for `server`, `team`,
  and `pack`, mirroring the existing bot ladder (25/50/100/200 votes ->
  2/5/10/20 cents). Redemption itself (`POST
  /{target_type}/{target_id}/votes/credits`) was already fully generic
  code-wise; there just weren't any `vote_credit_tiers` rows for anything
  but bot, so every non-bot "convert votes" flow -- including the
  already-shipped server one -- silently computed 0 credits regardless of
  real vote count. Found while building the team/pack credits UI in
  Omniplex; fixes server too, which predates that work.

### Changed

- `GetPublicShopCoupons` now filters out coupons that aren't usable, have
  expired, or have already hit their max-use count -- a public "available
  offers" listing built on it (Omniplex) showing a dead coupon as if it
  were live would be actively misleading, not just harmlessly stale.
- `EntityVoteInfo`'s `vote_credits` flag (advisory -- "does this target
  type support vote credits") now reports `true` for `team`/`pack` too,
  matching the tiers added above. Was hardcoded false for both regardless
  of whether tiers existed.

### Fixed

- `GET /users/{id}`'s `user_bots` only ever queried bots owned directly
  (the `owner` column) -- a bot transferred to a team (`owner` cleared,
  `team_owner` set) vanished from it entirely for every member of that
  team, same bug `user_servers` would have if `GetIndexServersByTeamMembership`
  didn't already exist for servers. Added the missing
  `GetIndexBotsByTeamMembership` query and unioned it in, same as servers.
  Directly affects Omniplex's shop page, which sources its bot/server
  picker from this field.

## [1.7.1] - 2026-08-30

### Added

- `POST /list/search` now covers teams and packs, not just bots and
  servers. New `SearchTeamsPublic`/`SearchPacksPublic` queries back the
  `team`/`pack` target types, with the same query/tag-filter/vote-range
  shape as bots and servers already had (packs sort by creation date
  rather than votes -- pack vote counts were never kept in a materialized
  column the way bots/servers/teams are, so there's nothing cheap to sort
  by). Arcadia's staff search already covered team/pack; the public
  endpoint just hadn't caught up.
- Shop coupons can now actually be redeemed. `POST
  /{target_type}/{target_id}/shop/purchase` takes an optional
  `coupon_code`, checked against usability, expiry, applicable items,
  target types, allowed users, a global max-uses cap, and a per-target
  reuse cooldown (new `shop_coupon_redemptions` table, migrated onto a
  `shop_coupons` table that -- turns out -- never had a primary key
  either, added in the same migration). A coupon either overrides the
  item's cost or zeroes it out entirely, matching `ShopCoupon.Cents`'s own
  doc comment. The catalog/admin side of coupons has existed for a while;
  there was just nowhere to spend one until now. Deliberately not
  enforced: the coupon's `requirements` field, whose semantics were never
  defined anywhere in the codebase -- flagged rather than guessed at.

### Changed

- `search_list` (`POST /list/search`) was the last real holdout from the
  sqlc migration -- its dynamic, reflection-built column lists and
  Go-templated `WHERE`/`JOIN` clauses have been replaced with static sqlc
  queries (`SearchBotsPublic`, `SearchServersPublic`), using the same
  "always bind the arg, let an empty/zero value no-op the clause" trick
  the new team/pack search queries above needed. Verified against prod:
  identical result sets for every query shape the old template supported,
  run side by side.
- A purchase's logged `cents` (`shop_purchases`) now reflects what was
  actually charged after a coupon discount, not the item's list price.

### Fixed

- `SearchTeamsQueue` (Arcadia's staff search) compared `teams.id` (uuid)
  directly against its search-box input with no cast -- harmless for a
  pasted team ID, but Postgres rejects the bind outright for any other
  text before the query's own `OR name ILIKE ...` branch ever gets a
  chance to match, so a staff member typing a plain name search would
  500. Cast to `::text` on both sides, matching the fix already applied
  to `RecomputeApproximateVotes`'s equivalent join in 1.7.0.

## [1.7.0] - 2026-08-29

### Added

- A real, versioned schema migration tool: `cmd/migrate` (`go run
  ./cmd/migrate up|status|create <name>`), backed by
  [goose](https://github.com/pressly/goose). It tracks applied migrations
  in a `goose_db_version` table and connects using the same `config.yaml`
  DSN the rest of Popplio reads, instead of a separate hardcoded one. New
  migrations live under `db/migrations/` as timestamp-prefixed
  `-- +goose Up`/`Down` `.sql` files; applying them is a deliberate manual
  step (`make migrate` or the command above), never wired into a deploy.
  See `db/migrations/README.md`.

### Changed

- Every raw `pgx` SQL call in the module (`state.Pool.Query`/`.Exec`,
  `pgx.CollectRows`/`RowToStructByName`) has been converted to type-safe
  [sqlc](https://sqlc.dev)-generated code (`db/queries/*.sql` compiled to
  `db/*.sql.go`), across `routes/`, `arcadia/` (`rpc`, `impls`, `tasks`,
  `bot`, `panel`), `notifications/`, `webhooks/`, `teams/`, `apps/`,
  `captcha/`, `seo/`, `moderation/`, `validators/`, and `infernoplex`. A
  handful of call sites can't be modeled by sqlc's static query analysis
  and stay raw pgx on purpose, each documented in place: `search_list`'s
  template-built search SQL, two `pg_catalog`/`information_schema`
  dispatchers (`create_data_task`'s table/column and foreign-key
  discovery), one import-cycle case in `perms/staff.go`, one
  runtime-discovered-table cleaner (`arcadia/tasks/cleaners.go`), and one
  startup DDL statement (`arcadia/panel/server.go`'s
  `CREATE TABLE IF NOT EXISTS`).
- `exp/`'s one-off SQL scripts have been retired. Every one still relevant
  was ported into a proper goose migration under `db/migrations/` first;
  the folder itself has been removed. See `db/migrations/README.md` for
  where to find the originals if you need the history.
- Housekeeping pass over the sqlc migration's output: six pairs of
  `db/queries/*.sql` entries turned out to be byte-identical SQL under two
  different names (e.g. `CheckBadgeExists`/`CountBadgeByID`,
  `ActivateBotPremium`/`ApplyBotPremiumDays`) -- each pair collapsed into
  one, callers repointed. Dropped a dead unexported method
  (`perms.Set.contains`, fully superseded by `Has`) and the three leftover
  `golang.org/x/exp/slices` imports (stdlib `slices` has covered this
  since Go 1.21; everywhere else in the module already used it), which let
  `go mod tidy` drop the now-unused direct dependency. `data/seed.iblseed`
  -- a generated-and-forgotten artifact nothing in this repo reads -- is
  also gone and now gitignored.
- `assets.CheckBot` (bot add/lookup) now tries Discord's own public
  application RPC endpoint first and only falls back to JAPI.rest as a
  last resort, instead of the other way around -- JAPI going down or
  rate-limiting us used to take bot submission/lookup down with it, for
  data Discord already hands out for free. The JAPI request also now
  sends a descriptive `User-Agent` (confirmed with JAPI's own owner that
  requests work fine without an API key as long as one is set; the lack
  of one was likely why our traffic looked anonymous/abusive to it).
  `DiscordBotMeta.fallback` now means the reverse of what it used to:
  `true` if JAPI was needed, not RPC. One real cost: guild count,
  suggested description/tags, and flags only come from JAPI, so a bot
  added via the RPC-only happy path won't have them prefilled at
  submission time -- guild count corrects itself on the bot's first
  `POST /bots/{id}/stats`, and description/tags were always just
  submitter-editable suggestions, not stored as-is.

### Fixed

- Several Arcadia panel write paths (shop items, shop item benefits, shop
  coupons, badges, staff templates, staff disciplinary types, staff
  positions, blog entries, changelog entries) could silently write SQL
  `NULL` into a `NOT NULL` array column whenever a client omitted or
  nulled an array field entirely -- e.g. creating a changelog entry with no
  `"fixed"` key -- because the underlying `pgx` driver encodes a nil Go
  slice as `NULL`, not `{}`, regardless of whether the call went through
  raw SQL or sqlc. Every affected insert/update path now defaults a nil
  slice to empty before it reaches the query.
- `drivers.Send` (`webhooks/core/drivers/core.go`) pushed a "Webhook Send
  Failed" alert to both the in-site bell and push notifications for
  *any* non-nil error from `sender.Send`, including `ErrNoWebhooks`. Most
  bots simply don't have a rewards webhook configured, so this fired on
  practically every vote/review/team-edit event for them, misreporting a
  routine "nothing to deliver to" as a failure. Now excluded via
  `errors.Is(err, sender.ErrNoWebhooks)`, matching how `PullPending`
  already treated the same case.

## [1.6.0] - 2026-08-27

### Added

- Per-category notification preferences: a new `user_notification_prefs`
  table and `types.AlertCategory` (8 topic-level categories -- votes,
  bot/server reviews, payments, shop, webhooks, staff applications,
  reports, account security) let a user mute a whole class of alert
  instead of it being all-or-nothing. `notifications.PushNotification` now
  checks the sender's preference before saving to the inbox or sending a
  push; no stored row for a (user, category) pair defaults to enabled, so
  existing users see no change until they explicitly mute something. New
  `GET`/`PATCH /users/{id}/notification-prefs`.
- ~24 staff/system actions that previously only posted to the Discord
  mod-log now also notify the affected user directly (in-site alert +
  push): bot/server approve, deny, unverify, certify/uncertify,
  claim/unclaim, vote ban/unban, single-entity vote reset, auto-unclaim,
  bot removal (deleted from Discord), premium removal, ban/unban sync, and
  staff application approve/deny. Report resolution is new entirely --
  previously a resolved/dismissed report had zero visibility anywhere,
  including the mod-log.
- `arcadia/impls.NotifyOwners`, a small best-effort helper (fetch an
  entity's managers, send each one an alert, log-and-continue on failure)
  used by all of the above instead of each call site re-implementing the
  same loop.

### Fixed

- Vote reminders never actually reached a user's in-site notification bell
  -- `votereminders/vote_reminders.go` set `NoSave: true` on every
  reminder ("spammy, fills the db"), which skipped the `alerts` table
  insert entirely. Removed; a user who finds reminders noisy can now mute
  the `votes` category instead of them being silently broken for everyone.
- The monthly automated vote reset (`VoteResetter`) voided the underlying
  `entity_votes` rows but never recalculated bots/servers/teams' cached
  `approximate_votes` column, which is what the public listings actually
  sort and display by. Any entity that hadn't been voted for *since* a
  reset kept showing its stale pre-reset total indefinitely. The reset now
  recomputes every entity's count from what's left in `entity_votes` in
  the same transaction.
- `webhooks/core/drivers/core.go`'s failed-webhook-send alert had the
  title "Webhook Send Successful!" on an `AlertTypeError` alert -- a
  copy/paste bug. Now reads "Webhook Send Failed".
- `dependabot_alert.go`'s dismissal-reason truncation used `+=` instead of
  `=`, which duplicated the text instead of shortening it whenever it
  exceeded the field limit.
- Every existing `PushNotification` call site was missing a click-through
  `URL` (shop purchases, vote credit redemptions, premium/payment alerts,
  webhook alerts); most now link somewhere useful instead of being a dead
  end in the inbox.

## [1.5.0] - 2026-08-24

### Added

- Guild moderation commands for Arcadia, restricted to the main and testing
  servers: `/purge` (bulk-delete up to 100 messages, optionally filtered to
  one user), `/lock`/`/unlock` (deny or restore `@everyone`'s Send Messages
  on a channel), and `/modlogs` (look up a member's recent moderation
  history). Every kick/ban/timeout/warn/purge/lock/unlock action, plus
  auto-mod actions, is now recorded in a new `mod_cases` table so it's
  queryable rather than living only in the mod-log Discord channel.
- Passive auto-moderation for Arcadia (spam, invite links, mass mentions),
  off by default behind a new `arcadia.auto_mod` config flag. Deletes the
  offending message, DMs the author, and logs the action attributed to the
  bot itself rather than a staff member.
- Three new staff permissions backing the above: `purge_messages`,
  `lock_channels`, `view_mod_cases`.
- A curated, staff-authored changelog system covering both Popplio and
  Omniplex: `GET /changelogs/@all` (public, optionally filtered by
  `?project=popplio|omniplex`) and a full create/update/delete/list
  implementation behind the Arcadia panel's `UpdateChangelog` action, gated
  by a new `manage_changelog` permission. This replaces an earlier draft
  (`changelogs` table + `UpdateChangelog` DTOs) that was scaffolded but
  deliberately left as a hard 403 stub — the table now has its own
  `project`/`itag`/`published` columns so Popplio and Omniplex entries can
  coexist instead of colliding on a bare `version` primary key.

### Fixed

- `Keel/ratelimit`'s exceeded check ran against the pre-increment request
  count, so a `MaxRequests: N` bucket actually let `N+1` requests through
  before blocking — the check now runs after incrementing, against the
  count that includes the current request. Affects every rate-limited
  route across the API; limits are now exactly as configured rather than
  one request looser.

### Changed

- Popplio's dependency on the shared library formerly known as `eureka`
  moved from `github.com/infinitybotlist/eureka` to `github.com/PlexiOSS/Keel`
  (all ~210 files' imports rewritten, `go.mod` updated).
- `popplio/db.GetCols` — a byte-for-byte duplicate of the `Keel/dbutil.GetCols`
  the package was already extracted into — deleted. All 56 call sites now
  use `Keel/dbutil` directly, finishing a migration that only relocated the
  code the first time around.
- `validators.Pointer`/`TruePtr`/`FalsePtr`, `validators.EncodeUUID`, and
  `validators.IsNonProdFrontend` had no Omniplex-specific knowledge in them
  at all — moved to new `Keel/ptr`, `Keel/uuidutil`, and `Keel/urlutil`
  packages respectively (the last renamed to `urlutil.DifferentHost`, since
  a shared library shouldn't have "frontend" in a function name). Also
  found and collapsed a second, independent duplicate of the UUID-formatting
  logic in `arcadia/impls.UUIDString`, which now calls `uuidutil.Encode`
  instead of re-implementing the same hex-dashing by hand.
- `validators.NormalizeTargetType` and the `captcha` package were
  considered for the same move and deliberately left in Popplio: the
  former's case list (`bots`→`bot`, etc.) *is* Omniplex's domain
  vocabulary, and the latter is genuinely coupled to `popplio/state` in
  three places (config, Redis, a direct Postgres query) — moving it would
  mean decoupling it first, not just relocating a file.

### Fixed

- `TestGoldenPanelStrings` only ever scanned `arcadia/panel`'s source for its
  frozen strings, but `identityExpired`/`sessionNotActive` are values
  defined in `arcadia/impls` (`ErrIdentityExpired`/`ErrSessionNotActive`)
  and only surfaced *through* panel responses — so the test was failing on
  every run, permanently, checking the wrong directory for two of its own
  entries. `assertContains` now takes multiple dirs.
- Shop coupon validation (`validateCoupon` in `arcadia/panel/ops_shop_coupons.go`)
  rejected a `null` `max_uses`/`reuse_wait_duration`/`expiry` as if it were
  `0`, making the "unlimited uses" and "never expires" cases the DTOs'
  own doc comments describe impossible to actually create. `null` is now
  treated as "no constraint"; a present-but-non-positive value is still
  rejected. See `arcadia/CONFORMANCE.md` #2.
- The bot-path `Unverify` RPC action's mod-log embed had a field with an
  empty name, which Discord's API rejects — the embed post failed and the
  whole call errored out *after* the bot had already been flipped back to
  `pending`, with no rollback. The field now has a real name (`"Bot"`,
  matching the server path's existing `"Server"` field). See
  `arcadia/CONFORMANCE.md` #9.
- Creating a vote credit tier that landed on an already-doubly-occupied
  position 500'd with a raw Postgres constraint violation (this was live —
  production's existing tiers at positions 1-3 made creating a new tier at
  position 1 fail outright). `dedupTierPositions` is now a single set-based
  `UPDATE ... SET position = position + 1 WHERE position >= $1`, replacing
  an iterative loop whose cursor skipped past positions it had just written
  to. See `arcadia/CONFORMANCE.md` #16.
- A staff position's "testing" corresponding-role (main-guild-adjacent role
  auto-grant/revoke) validated but silently never synced — only `main` and
  `staff` were handled. Added the missing case. See `arcadia/CONFORMANCE.md`
  #8.
- `/list/staff-templates` was registered with `OpId: "get_partners"` — a
  copy-paste collision with the actual `/list/partners` route above it.
  Now `get_staff_templates`.
- `PopplioStaff` (the staff panel's signed reverse proxy into Popplio's own
  `/staff/*` API) sent the request path as `X-Forwarded-For` instead of the
  caller's actual IP. Confirmed this wasn't cosmetic: `create_data_task`
  reads that header as the requester's IP for a GDPR audit record, and the
  proxy isn't restricted to `/staff/*` paths, so a staff member's real IP
  could end up silently replaced with a URL path in that record. The
  panel's request context now carries the real caller IP through to the
  proxy. See `arcadia/CONFORMANCE.md` #5.
- `Authorize/Begin` built the Discord OAuth login URL by interpolating
  `redirect_url` raw — neither validated nor URL-encoded. The actual
  token-exchange step already checked it against an allow-list, so this
  couldn't complete a session with a bad value, but it could still hand
  back a malformed or attacker-chosen login URL. Now validates against the
  same allow-list up front and URL-encodes the value. See
  `arcadia/CONFORMANCE.md` #13.
- `topreviewer_sync` stripped the top-reviewer Discord role from everyone
  weekly and never regranted it (hardcoded `LIMIT 0` instead of the `3` the
  `/refresh` command already used for the same query) — the role has been
  permanently empty since this job started running. Now regrants to the
  actual top 3. See `arcadia/CONFORMANCE.md` #3.
- Disciplinary type `created_at` was populated from the most recent
  disciplinary action against a staff member, not from when the type
  itself was created — confirmed unused in Omniplex's admin UI, so this
  was silently wrong with no observable effect until now. Now selects the
  type's own `created_at`. See `arcadia/CONFORMANCE.md` #7.
- A dead `testbot` branch in `Claim` (unreachable — the check above it
  already rejects anything that isn't `pending`) removed. `Unclaim`'s live
  `testbot` check, which runs before its `pending` check, is untouched.
  See `arcadia/CONFORMANCE.md` #6.
- A batch of frozen error/embed strings, now that Omniplex's own admin
  panel is the only consumer of this wire format: `"[neeed to delete
  position]"` → `"[need to delete position]"`; the `Invalid OTP
  Entered`/`entered` casing mismatch between `ResetMfaTotp` and
  `ActivateSession` unified to lowercase; `bot_whitelist` permission
  messages switched from `(parentheses)` to `[brackets]` to match every
  other permission message; a batch of bare-leading-space error strings
  restored to interpolate the target id again (e.g. `" does not exist"` →
  `"%q does not exist"`); and every leading-space mod-log embed title
  (`" Claimed!"`, `" Approved!"`, `" Force Deleted!"`, and five others)
  had the stray space dropped. `"__ Unverified For Futher Review!__"` was
  deliberately left alone — same class of typo, but out of scope for this
  pass. See `arcadia/CONFORMANCE.md` #11, #12.

### Removed

- `GET /list/current-status` and its Instatus/UptimeRobot proxying
  (`state.Config.Sites.Instatus`, `state.Config.Meta.UptimeRobotROAPIKey`,
  `types.StatusDocs`). Confirmed unused anywhere in Omniplex's frontend —
  status reporting now lives entirely on the dedicated
  `status.omniplex.gg` instance, so this endpoint (and the two external
  status-provider integrations behind it) was dead weight.
- `GET /list/team` (`get_list_team`) and the now-unused `types.StaffTeam`.
  Its own doc comment admitted it was "currently broken and does not
  handle permissions yet" — fully superseded by this session's `GET
  /staff/team`, which Omniplex's team page actually uses.

## [1.4.0] - 2026-08-24

### Added

- `GET /staff/team` — a public, unauthenticated staff roster for the
  website's team page: user ID, username, avatar, and the position(s) each
  member holds (name/icon/rank only). Unlike everything else under
  `/staff/*` this deliberately skips auth, same reasoning as the existing
  `/staff/meta/permissions`: it's who holds a title, not what any
  individual permission grant is. Permissions, disciplinaries, and
  sync/security metadata never leave the staff panel.
- `GET /list/stats` now reports server counts alongside the existing bot
  ones: `total_servers`, `total_approved_servers`,
  `total_certified_servers`, `total_pending_servers`,
  `total_denied_servers`, `total_vote_banned_servers`. The endpoint had
  bot review-pipeline and vote-ban counts from the start but never grew a
  server equivalent, so the moderation transparency page could only ever
  show half the picture.
- A `noxss` field validator (`state/xss.go`, registered alongside the
  existing `nonvulgar` one), rejecting `<script>`, `javascript:`/
  `vbscript:` URLs, inline event handlers (`onerror=`, `onload=`, ...),
  `<svg>`/`<object>`/`<embed>`/`<meta>`, and legacy CSS `expression()`
  injection. Applied to every large free-text field that reaches the API:
  `CreateBot`/`BotSettingsUpdate` and `CreateServer`/
  `ServerSettingsUpdate` (`short`/`long`), and `CreateReview`/`EditReview`
  (`content`). Deliberately does not flag `<iframe>` Omniplex's own
  markdown renderer allows it by explicit product decision for bot-owner
  embeds (see its `sanitizeSchema.ts`), so rejecting it here would just
  reject legitimate submissions. Also extended to `CreateEditTeam.Short`,
  packs' `CreatePack.Short`/`PatchPack.Short`, and user profile bios
  (`PATCH /users/{id}` inlines a `state.ContainsSuspiciousMarkup` check
  instead, since that route does not validate its payload via struct
  tags).
- Staff can now spotlight a bot or server on the home page, alongside the
  existing Feature action, two new RPC methods, `SpotlightAdd`/
  `SpotlightRemove` (`arcadia/rpc/spotlight.go`), mirroring `FeatureAdd`/
  `FeatureRemove` exactly (same `time_period_hours` field, same
  `GREATEST(COALESCE(...), NOW()) + make_interval(...)` extension
  behaviour), gated by the same `feature_entities` permission. Backed by
  a new `spotlighted_until` column on `bots`/`servers`
  (`exp/spotlight_columns.sql`, not auto-applied) and exposed on both
  list-index endpoints (`GET /bots`, `GET /servers`) as a new `spotlight`
  array, the same way `featured` already was.
- `moderation.FileAutoReport` a newly-flagged bot/server now gets an
  actual report filed (reason `tos_violation`, a fixed
  `system:moderation` reporter) so it lands in the staff review queue,
  not just a passive `moderation_flagged` column nobody's necessarily
  looking at. Wired into both the submission-time check (`add_bot`,
  `add_server`) and the new `moderation_scan` background task below.
- New `moderation_scan` background task (`bgtasks/moderation_scan.go`,
  every 30 minutes, batched to 25 bots + 25 servers per run): re-runs
  OpenAI moderation against listed bots/servers on a rolling ~7-day
  cadence, not just at submission time. Catches entries that predate
  `moderation_flagged` entirely and edits made after initial submission
  (`PATCH .../settings` never re-runs the check). Tracked via a new
  `moderation_checked_at` column (`exp/moderation_scan_columns.sql`, not
  auto-applied — run with `psql "$DATABASE_URL" -f
  exp/moderation_scan_columns.sql`); no-ops entirely when
  `meta.openai_api_key` is unset, same as the existing check.

### Fixed

- `PremiumAdd`/`FeatureAdd`/`SpotlightAdd`'s `time_period_hours` field
  metadata claimed "Format: X years/days/hours" — the field has only ever
  accepted a bare hour count (`int32`), never a compound duration string,
  so that placeholder was actively misleading staff filling out the field
  through the panel. Placeholder now reads "Duration, in hours (e.g. 720
  for 30 days)".

### Changed

- Popplio no longer has a build-time `staging`/`beta`/`dev` environment —
  `config.Differs[T]` and the `//go:embed current-env` mechanism are gone,
  and every config value that used to need up to four variants
  (`token`, `redis_url`, port numbers, PayPal/Stripe keys, etc.) is now a
  single plain value. Staging and beta/dev instances of Popplio, Arcadia,
  and Infernoplex were deprecated; keeping a whole generic config layer
  and a dozen `if CurrentEnv == ...` branches around for environments that
  no longer exist was just a trap for the next person who forgets to keep
  all four variants of a secret in sync. The "only run this in prod"
  guards (Discord presence, background tasks, vote reminders) are gone
  too — there's only one deployment now, so they were permanently true.
  PayPal always talks to the live API rather than switching to sandbox
  outside of prod, for the same reason.
- The Bug Hunter-only sign-in restriction (previously scoped to
  `CurrentEnv == beta || staging`) is now scoped by which frontend is
  actually calling, not which binary is running: it compares the OAuth
  `redirect_uri` (at login) or the request's `Origin` header (on every
  authenticated request) against the configured production frontend.
  Multiple frontends (a staging/beta site, local dev) can still point at
  the one shared backend; this is what actually varies per request now
  that the backend itself doesn't.

### Removed

- The `staging`/`beta`/`dev` config sections (and `config.Differs[T]`
  itself) — see Changed above. `config.yaml` now takes a single value per
  field instead of nesting one per environment; `config.yaml.sample`
  regenerates itself flat the next time the binary starts.
- `validators.StagingCheckSensitive`, the payments perk-gating check that
  used to require a special staff permission to touch PayPal/Stripe
  outside of prod (since staging used test keys). There's only one set of
  keys now, so the thing it was protecting against can't happen anymore.

## [1.3.3] - 2026-08-22

### Added

- Infernoplex and the staff bot (Arcadia) now set a Discord presence on
  connect — "Watching Omniplex servers" and "Watching the review queue"
  respectively. Neither had one before (default "Playing nothing"). Gated
  to the prod instance only, same as Popplio's main bot: staging/beta/dev
  never broadcast a live-looking presence, whether from a shared token or a
  local checkout pointed at real credentials.
- NSFW compliance signal on `Server` (`discord_nsfw_level`, `nsfw_channel_count`)
  and the review queue/search panel ops that expose it — reviewers previously
  had to join a server and look around by hand to check "Server: NSFW Content
  Not Gated." Infernoplex's periodic `syncServerMeta` task now also fetches
  the guild's channel list each cycle and counts how many have Discord's own
  age-restricted flag set, plus the guild's own NSFW classification
  (`guild.NSFWLevel`). New columns via `exp/server_nsfw_compliance.sql` (not
  auto-applied — run with `psql "$DATABASE_URL" -f
  exp/server_nsfw_compliance.sql`).
- New `moderation` package wraps OpenAI's moderation endpoint
  (`omni-moderation-latest`, free to call). `POST /bots` and `POST /servers`
  now run the submitted short/long description through it right after
  insert and store the result on `moderation_flagged`/
  `moderation_categories`, surfaced on the review queue/search panel ops the
  same way the NSFW compliance fields are — a reviewer signal, not an
  auto-reject; nothing reads these columns to gate anything. Configured via
  a new `meta.openai_api_key` config value; moderation is silently skipped
  when it's unset, so this is a no-op until a key is added. New columns via
  `exp/moderation_columns.sql` (not auto-applied — run with `psql
  "$DATABASE_URL" -f exp/moderation_columns.sql`).

- Full CRUD for the staff-template catalog (pre-built answers staff pick
  from when approving/denying a bot or server review) via a new
  `UpdateStaffTemplates` panel op, gated by a new `manage_templates`
  permission — previously the only way to add or edit one was a manual DB
  insert, since nothing in Popplio or Arcadia wrote to `staff_templates`
  at all.
- `POST /servers/stats` — lets a server self-report `total_members`/
  `online_members` via a server-scoped API token, the same way bots have
  long been able to via `POST /bots/stats`. Posting at all flips a new
  `stats_self_managed` flag on, which tells Infernoplex's periodic
  `syncServerMeta` task to stop overwriting those two fields for that
  server (it still keeps the icon in sync either way) — otherwise the
  automatic sync and a server's own self-reports would just fight each
  other every 30 minutes.
- Staff review templates now carry an `entity_type` (`bot` or `server`)
  column — `GET /list/staff-templates` previously only ever documented
  itself as "used for reviewing bots," with no way to scope a template to
  servers at all. Existing rows default to `bot`. Filter with
  `?entity_type=bot` or `?entity_type=server`; omit it for both.
- `CertifyAdd`/`CertifyRemove` and `PremiumAdd`/`PremiumRemove` (staff RPC
  actions) now support servers as well as bots — same pattern
  `Claim`/`Approve`/`Deny`/`Unverify` already established: the handler
  branches on `TargetType` to a `*Server` counterpart. Certifying a server
  moves it to `type = 'certified'`; uncertifying returns it to `approved`,
  same as bots.
- A new `FeatureAdd`/`FeatureRemove` staff RPC action (gated by a new
  `feature_entities` permission) lets staff put a bot or server in the home
  page's Featured section for a given time period, or pull it early —
  previously `featured_until` was only ever settable through a shop
  purchase (`routes/shop/assets/benefits.go`), with no staff override.
  Storage matches the shop path exactly (stacks with a bought featured
  slot instead of clobbering it), generalized across bots/servers via the
  same `table`/`idCol` pattern the shop benefits code already used.
- Added `@ci` struct annotations to `types/server.go` (`IndexServer`,
  `Server`, `CreateServer`). Every other entity's types file (`bot.go`,
  `pack.go`, `user.go`, etc.) has these, wiring it into
  `db_fields_check.py`'s struct-vs-schema validation — `server.go` was the
  one file that never got them, so a `servers` column drifting out of sync
  with its struct field (renamed, dropped, added and never wired up) would
  go uncaught by CI while every other entity was protected.
- New GIN indexes (`exp/search_gin_idx.sql`, not auto-applied run with
  `psql "$DATABASE_URL" -f exp/search_gin_idx.sql`) back `POST
  /list/search`: `bots_short_fts_idx`, `servers_name_fts_idx`,
  `servers_short_fts_idx` for the `short @@ $query` / `name @@ $query`
  full-text matches, and `servers_name_trgm_idx` (via `pg_trgm`) for
  `name ILIKE '%...%'`. None of the columns search queries against had an
  index before this, so every search request was a full sequential scan
  across every approved/certified bot and server, recomputing
  `to_tsvector()` per row on every call.

### Changed

- Renamed four staff permissions that gate bot *and* server actions but
  were named after bots only: `review_bots` → `review_entities`,
  `certify_bots` → `certify_entities`, `force_remove_bots` →
  `force_remove_entities`, and the `marker_bot_reviewer` marker →
  `marker_reviewer`. Functionally nothing changes — `review_entities`
  still gates the same `Claim`/`Unclaim`/`Approve`/`Deny`/`Unverify` RPC
  actions it always did, and `force_remove_entities` still covers packs
  too, same as before. `exp/rewrite/rename_reviewer_perms.sql` (not
  auto-applied — run with `psql "$DATABASE_URL" -f
  exp/rewrite/rename_reviewer_perms.sql`) renames the already-stored flat
  names in `staff_positions.perms`, `staff_members.perm_overrides`, and
  `staff_disciplinary_types.perm_limits`; safe to run more than once.

### Fixed

- `StaffMember` never exposed a way for the panel to know a viewer's actual
  seniority rank (`perms.StaffGrants.Rank()`) — an instance owner holding
  no explicit position had no way to be told apart from a regular staff
  member holding none, and the frontend derived a "lowest held index"
  from `positions` that put both at the same (locked-out-of-everything)
  end. New `rank` field on the API response, mirroring `Rank()` exactly
  (owners get `math.MinInt32`, no-position members get `NoRank`).
- `/health/bots`, `/health/servers`, `/health/packs`, `/health/blogs`,
  `/health/search`, `/health/auth`, `/health/tickets`, and
  `/health/staff-panel` all used `SELECT EXISTS(SELECT 1 FROM <table>
  LIMIT 1)` as their check — a row-presence test, not a health check. A
  perfectly healthy table with zero rows (an empty `blogs` table, a fresh
  instance with no tickets yet) reported as DOWN. Now checks the table
  exists in `information_schema.tables` instead, which still catches a
  genuinely missing/unmigrated table without requiring it to have data.

### Removed

- Infernoplex's `/setup` command — the website's `PUT /servers` (Add
  Server) already resolves a server from its invite link without needing
  the tracking bot present at all, and the staff review pipeline now
  provides the ownership-verification step `/setup`'s `AdminOnly` check
  used to be the only thing doing. Team creation and server-record setup
  both already happen through the normal website flow.

## [1.3.2] - 2026-08-17

### Added

- Sign-in on beta and staging is now restricted to Bug Hunters (instance
  owners bypass it, so they can't lock themselves out of their own
  environment). Rejected cleanly at the OAuth login step
  (`checkBugHunterOnly`, same pattern as the existing `checkBanScope`),
  plus a matching global check in `api/uapi.go` for defense in depth on
  any session issued before this. Reads the same `users.bug_hunters`
  column `SpecRoleSync` already keeps in sync with the Bug Hunter Discord
  role no new sync mechanism, no new schema.

- Real account bans: a new `BanUser`/`UnbanUser` RPC action (new
  `ban_users` permission) sets `users.banned`, distinct from
  `AppBanUser`/`AppUnbanUser`'s much narrower `app_banned` flag. Nothing
  previously set the column at all despite `api/uapi.go` already rejecting
  every authenticated request from a banned user except sessions scoped
  `ban_exempt` a full account ban was completely unreachable through any
  staff action.
- `VoteBanAdd`/`VoteBanRemove` now support `Server`, `Team`, and `Pack` in
  addition to `Bot` (all four carry an identical `vote_banned` column).
  `ForceRemove` now supports `Server` and `Pack` in addition to `Bot`
  the `kick`/protected-bots behaviour stays bot-only, since neither has a
  "leave the guild" equivalent. This is the reports-can't-act-on-non-bot-
  content gap: reports against a server or pack had no staff action to
  take beyond bots.
- A generic badge system: a staff-managed catalog (`badges`) plus a
  flexible assignment table (`entity_badges`), so a new purely-decorative
  badge is a catalog row and an assignment from now on, not a new column,
  backend flag, and frontend branch every time. New `AssignBadge`/
  `UnassignBadge` RPC action (new `assign_badges` permission, works on
  User/Bot/Server/Team) reaches every entity type through the same
  Actions menu as every other staff action, and a new
  `GET /{target_type}/{target_id}/badges` public route reads them back.
  Deliberately separate from the *functional* badges already on
  users/bots/servers (premium, certified, developer, `bug_hunters` the
  last one specifically because it's synced from a Discord role by
  `SpecRoleSync`, not manually assigned, so it stays exactly as-is)
  new `manage_badges` permission gates the catalog itself.
- Bots can now document their own commands and post changelog/announcement
  entries, gated by the same `edit_bots` entity permission (owner or team)
  that already gates editing a bot's settings no new permission needed.
  `PUT /bots/{id}/commands` replaces the whole command list (same
  full-replace convention as `extra_links`); `POST`/`DELETE
  /bots/{id}/changelogs` append and remove individual entries (same
  convention as reviews). Both are public to read.

  - A new `Health` route group, `GET /health/*` — one endpoint per subsystem
  (database, API, bot/server/pack listings, blog, search, Discord auth,
  tickets, staff panel, plus Infernoplex's and Arcadia's separate Discord
  gateway connections), each returning a bare 200/503 with no body to
  parse. Built for the new external status page (omni-status) to point one
  uptime monitor at each endpoint — mirrors what Omniplex's own
  `/about/status` page already self-probes client-side, done server-side
  instead so an outside monitor doesn't need API-client internals.

- Servers now go through the same staff review pipeline bots already do:
  `PUT /servers` previously defaulted straight to `type = 'approved'`, so
  every server went live immediately with zero review, unlike bots
  (`type` default flipped to `pending` via `exp/serverreview.sql`, existing
  rows untouched). `Claim`/`Unclaim`/`Approve`/`Deny`/`Unverify` now support
  `Server` alongside `Bot` (shares the existing `review_bots` permission,
  same precedent as `VoteBanAdd`/`ForceRemove` already covering multiple
  target types under one permission), and a new `ServerQueue` panel op
  mirrors `BotQueue`. No server equivalent of the bot-approval Discord
  role auto-grant exists, so `Approve` stops at the state transition and
  mod-log post for servers.
- Some `Bot Reviews`/`Users & Votes` staff permissions (`transfer_bots`,
  `force_remove_bots`, `manage_premium`, `manage_votes`, `ban_voters`)
  reclassified under a new `Content Management` category — these act on
  listed entities (transferring, deleting, granting perks, resetting
  votes, vote-banning), not on the review queue or on user accounts, so
  they read oddly grouped with either.
- `GET /servers/@emojis/flat` and `GET /servers/@stickers/flat` — unnest
  every opted-in server's emojis/stickers into one flat, item-level-paginated
  list (60/page) instead of `GET /servers/@emojis`'s one-page-per-server
  shape, for a cross-server browse page that doesn't grow one section per
  server as more servers opt in.

### Fixed

- Animated emojis were synced with a permanently-static CDN URL despite
  `animated: true` being stored correctly — `disgo`'s `Emoji.URL()`
  always defaults to PNG regardless of the emoji's animated flag (unlike
  `Sticker.URL()`, which already inferred the right format from
  `FormatType`). `serversync.go` now explicitly requests GIF format when
  `Animated` is true.
- Server `total_members`/`online_members` were only ever set once, at
  `/setup` time, and never refreshed — there's no periodic member-count
  sync, and the bot deliberately doesn't hold the privileged Server
  Members intent (see `TeamCleanup`'s doc comment), so the gateway's
  cached guild object never gets live count updates either. The existing
  30-minute server-sync task (renamed `syncAvatars` → `syncServerMeta`)
  now also REST-polls `GetGuild(id, true)` per server and updates both
  counts, the same way `/setup` originally got them.

## [1.3.1] - 2026-08-16

### Added

- Arcadia's `SearchEntitys` panel action now supports `Pack`, `Team`, and
  `User` in addition to the existing `Bot`/`Server` previously any other
  target type 501'd. Backs the admin Search page now covering every
  entity type and its respective staff actions instead of just bots and
  servers. New `PartialPack`/`PartialTeam`/`PartialUser` variants added to
  the `PartialEntity` wire union (additive existing `Bot`/`Server`
  consumers are unaffected).
- `GET /users/{id}` now returns `user_servers`, the servers owned by any
  team the user is on (mirroring the existing `user_bots`/`user_packs`
  resolution). Servers have no direct `owner` column team ownership is
  the only path, same as `GetOwnedBy`'s server branch. Public user
  profiles previously had no way to show a user's servers at all.
- The staff bot gained guild moderation commands `/kick`, `/ban`,
  `/timeout`, and `/warn` (new `moderate_guild`/`warn_users` permissions),
  plus self-serve `/kb`, `/ticket`, and `/staffinfo` for pointing users at
  the Knowledge Base, ticket support, and the staff hierarchy without
  retyping the same links. Every moderation command refuses to act on a
  target who is themselves staff at a rank equal to or more senior than
  the caller's own (`perms.LoadStaff(...).Rank()`) `staff_positions` is
  the same hierarchy Discord role assignments already sync into via
  `StaffResync`, so this is one hierarchy check, not a separate Discord
  one and a separate Omniplex one.

### Fixed

- Two Discord embeds in `routes/staff/endpoints/manage_app/route.go`
  ("Application Approved"/"Application Denied") still linked to the
  deprecated SvelteKit panel (`Sites.Panel` + `/panel/apps`) after the
  equivalent submission-time embed was already fixed to point at
  Omniplex's `/admin/applications` — these two were missed in that pass.
- `StaffResync` (`arcadia/tasks/staffresync.go`) inserted a new row into
  `staff_members` before checking whether that user had a `users` row yet.
  `staff_members.user_id` has a foreign key into `users`, so any staff
  member who held a staff role in Discord but had never actually logged
  into Omniplex made every resync run fail with a `staff_members_user_id_fkey`
  violation, repeating on every scheduled run until fixed. The
  ensure-`users` row exists check now runs before the `staff_members`
  insert/update instead of after it.

## [1.3.0] - 2026-08-15

### Added

- Certification approval (`reviewLogicCert`/`reviewLogicCertServer` in
  `apps/logic.go`) now automatically grants `BotDeveloper`/
  `CertifiedDeveloper` roles to a certified bot's owner(s), or every
  member of a certified server's owning team, provided they're already in
  the main guild — previously only the bot's own `CertBot` role was
  granted, and owners had to know to run `ibb!getbotroles` themselves.
- `GET /list/stats` gained `total_banned_users` and `total_vote_banned_bots`,
  aggregate `COUNT(*)` queries over the existing `banned`/`vote_banned`
  columns, for the Moderation Transparency page's new "Platform safety"
  section. Public, no PII — same pattern as the existing report-stats
  endpoint.

### Fixed

- A Discord API failure while granting a bot's own `CertBot` role during
  certification (e.g. the bot not currently being in the server) used to
  hard-fail the entire review, leaving the application stuck "pending"
  even though `bots.type` had already been committed as `certified`
  separately. It's now logged as a warning instead of aborting the review.
- `GetOwnedBy` (`arcadia/impls/entities.go`) only checked team ownership
  for bots, silently missing bots owned directly — meaning a direct owner
  got "you don't own any bots" from `/getbotroles` even when they
  genuinely did. Added the missing `OR owner = $1` branch.
- `/staff/tickets?open=true` was 500ing: 8 legacy ticket rows had
  `messages` stored as a JSON object instead of an array, and
  `pgx.RowToStructByName` fails the whole result-set scan on a single
  row's type mismatch. Normalized the affected rows; `create_ticket`
  already always writes a real array, so this was legacy data, not a
  recurring bug.

## [1.2.2] - 2026-08-15

### Fixed

- `arcadia/panel/ops_proxy.go`'s `popplioStaff` proxy (backs the new
  Applications admin page) rejected every request with a misleading
  "Path must start with /" error in production, even for a
  correctly-formed path like `/staff/apps`. Root cause was in
  `safeJoinPopplio` (`arcadia/panel/paths.go`): beyond the real security
  boundary (same scheme+host as Popplio's own API base), it also enforced
  that the resolved path stay under the *same path prefix* as the
  configured base URL — which rejects any legitimate root-level target
  whenever that base URL has a non-root path component. That check added
  no security beyond the origin check and only broke valid callers, so
  it's removed. Also stopped collapsing every `safeJoinPopplio` error into
  the same fixed string — the real error now surfaces, so a future failure
  here is diagnosable instead of misleading.
- `notifications.PushNotification`'s `NoSave` field was inverted from its
  own name/doc comment (`if notif.NoSave { INSERT }` — persisted only when
  told *not* to save). In effect, every "normal" alert (push-subscribe
  confirmation, reminder-set confirmation, payment-failure alerts) never
  reached a user's in-app alert inbox, only ever firing as a transient
  push notification — the one caller that explicitly opted out of saving
  (`vote_reminders.go`, "spammy, fills up the db quickly") was the only
  alert type that persisted. Condition is now `if !notif.NoSave`, matching
  what the field has always been named and documented to mean.
- The "New Application" Discord embed (`routes/apps/endpoints/create_app`)
  linked to the old SvelteKit panel (`Sites.Panel` + `/panel/apps`),
  superseded by Omniplex's own `/admin/applications` — link updated.

### Added

- `GET /staff/tickets` — every ticket platform-wide, gated on the existing
  `view_tickets` staff permission (optional `?open=true|false` filter,
  paginated). Staff could already view/reply/close/reopen any ticket via
  the existing owner-or-staff checks on `get_ticket`/
  `create_ticket_message`/`patch_ticket`, but had no way to find a ticket
  ID to act on in the first place — this closes that gap. Auth follows the
  same normal-user-session + in-handler permission check as the other
  ticket routes, not the legacy `staffpanel__authchain` system the
  Applications page uses.
- A user-facing confirmation alert (now that `PushNotification` actually
  persists them) at three points that previously gave zero in-app
  feedback on success — a purchase completing (`GivePerks`, alongside the
  existing staff-only mod-log post), a shop item purchase, and a vote
  credit redemption. All three are best-effort: the underlying change is
  already committed by the time the alert is sent, so a failed alert logs
  a warning rather than turning into an error response for something that
  actually succeeded.
- A new report being filed now posts to the staff-only `StaffLogs`
  Discord channel (target, type, reason — no reporter identity, consistent
  with reporter identity being staff-panel-only everywhere else). The
  equivalent "new application submitted" post already existed
  (`create_app` posts to the `Apps` channel with an `@Apps` role ping) —
  confirmed by reading the handler directly, not assumed.

## [1.2.1] - 2026-08-15

### Changed

- The Friday-Sunday double-vote weekend bonus (`votes.GetDoubleVote`) now
  pins its day-of-week check to UTC explicitly instead of relying on
  `time.Now()`'s implicit host-local timezone, so the boundary is the same
  instant regardless of what timezone the process happens to run in. Also
  added `VoteInfo.WeekendBonus`, a per-entity flag reporting whether the
  bonus is actively boosting that entity's `per_user`/`vote_time` right
  now — `false` for premium bots/servers even on a bonus weekend, since
  their flat premium cooldown already applies instead. Nothing in the API
  previously told callers when the bonus was live.
- Certification requirements loosened and diversified. The old rule
  required a bot to clear servers ≥100 **and** unique clicks ≥30 both,
  no exceptions. It's now an OR across three lowered bars (servers ≥50,
  unique clicks ≥15, **or** votes ≥50, the last one is new), plus a new but
  lenient 3-day minimum listed age. A bot excelling in one metric no
  longer gets rejected for not excelling in all of them.
- Servers can now be certified too a new "Server Certification"
  position on `/apps` (`extraLogicCertServer`/`reviewLogicCertServer` in
  `apps/logic.go`), using the same OR-of-three-metrics rule scaled to
  server stats (members ≥100 in place of bot servers ≥50). New
  `request_server_certification` permission. The "Certified" badge
  Omniplex has always been able to render for servers had no backend path
  that ever set it until now.
- Premium and Shop now work for servers, not just bots:
  - `CreatePerkData`/`PerkData` gained a `for_type` field (`"bot"` or
    `"server"`, defaults to `"bot"` if omitted) so Stripe/PayPal checkout
    and the booster-offer redemption can target either.
    `servers.premium` has been a real, displayed column with no purchase
    path behind it since it was added; now there is one.
  - Shop purchases (`POST /{target_type}/{target_id}/shop/purchase`) and
    all five benefit effects (`routes/shop/assets/benefits.go`) now
    branch on target type between `bots`/`servers` both tables carry
    identical benefit columns.
  - `servers` gained the same `boosted_until`/`featured_until`/
    `supporter_badge`/`vote_blitz_until` columns bots already had
    (`exp/serverbenefits.sql`), plus the matching read-side effects:
    boosted-first sort in `GET /servers/@all`, a `featured` category in
    `GET /servers/@index`, and a vote-blitz cooldown halving in
    `EntityVoteInfo`'s `"server"` case.

### Added

- `GET /list/stats` now includes `total_pending_bots` and
  `total_denied_bots`, so consumers can show a real approved/certified/
  pending/denied breakdown instead of inferring it from `total_bots` minus
  the listed count.
- `GET /staff/shop-purchases` the same shop-purchase data
  `GET /{target_type}/{target_id}/shop/purchases` already exposes
  publicly one entity at a time, but platform-wide and staff-gated
  (`view_shop`) for abuse/fraud monitoring. No frontend consumes this yet
  since the Arcadia panel UI isn't part of this repo it's ready for
  whenever that side wires it up.
- A standalone support ticket system. The existing `tickets` table/`Ticket`
  type were entirely Discord-channel-shaped (`channel_id`, `enc_key`) with
  no creation path anywhere in the codebase, not in the API, not in the
  Discord bot nothing has ever created a ticket in this codebase. Rather
  than build the Discord-integration side, tickets are now a plain web
  feature reusing the same table (`channel_id` left `""`, `enc_key` left
  null): `GET /tickets/topics` (a small hardcoded topic catalogue, same
  convention as `apps.Apps`), `POST`/`GET /users/{id}/tickets`,
  `POST /tickets/{id}/messages`, and `PATCH /tickets/{id}` to close/reopen
  (closing is open to the author or staff; reopening is staff-only, via a
  new `manage_tickets` permission). New message IDs are synthesized
  Discord-format snowflakes (`disgoorg/snowflake`'s `New`) purely so
  `GET /tickets/{id}`'s existing `snowflake.Parse`-based timestamp
  decoding keeps working unchanged for old and new tickets alike.
- Shop purchases actually do something now. `shop_items`/`shop_item_benefits`
  have had full staff CRUD via the Arcadia panel for a while, but nothing
  ever spent an entity's earned vote credits on one or defined what a
  benefit's effect even was. New:
  - `POST /{target_type}/{target_id}/shop/purchase` (bots only for now,
    gated on a new `buy_shop_items` entity permission) spends credits
    oldest-batch-first across `entity_vote_redeem_logs`, logs the purchase
    to a new `shop_purchases` table, and applies every benefit ID on the
    item that Popplio recognizes.
  - Five recognized benefit IDs, each with a real effect:
    `premium_days` (extends the bot's premium period, identical to the
    Stripe/PayPal path), `priority_boost` (new `boosted_until` column,
    sorts first in `/bots/@all`'s default order while active),
    `featured_slot` (new `featured_until` column, surfaces the bot in a
    new `featured` category on `/bots/@index`), `supporter_badge` (new
    permanent `supporter_badge` flag), and `vote_blitz` (new
    `vote_blitz_until` column, halves `EntityVoteInfo`'s vote-time
    cooldown while active). Unrecognized benefit IDs no-op rather than
    error, so staff can still catalogue purely descriptive/future
    benefits without breaking a purchase but an item with zero
    recognized benefits is rejected at purchase time rather than silently
    spending credits for nothing.
  - `GET /{target_type}/{target_id}/shop/purchases` purchase history,
    public, same transparency level as the existing vote-credit logs.
  - `exp/shopbenefits.sql` (schema: 4 new `bots` columns + the
    `shop_purchases` table) and `exp/shopbenefits_seed.sql` (optional
    starter catalog rows for the 5 benefits) the seed is just a
    starting point; the same rows can be created through the Arcadia
    panel instead.

### Changed

- Omniplex is now owned by NodeByte LTD. Remaining "Infinity Bot List" /
  "Infinity Development" copy left over from the old brand — application
  question text, the staff-denial DM, webhook docs, the RSS feed title
  and copyright line, the auth-log embed footer, and the `!delete`
  bot-command copy now reads "Omniplex" / "NodeByte LTD".

### Fixed

- The Gold premium plan granted ~365 hours (~15 days) of premium instead
  of a year `TimePeriod` was set in raw days while `GivePerks` applies
  it as hours. Bronze/Silver were already correct; Gold now multiplies by
  24 like they do.
- `POST /users/{id}/redeem-payment-offer?code=BOOSTPREMIUM` granted the
  perk successfully but then always fell through to a final `400 Invalid
  offer code` response regardless — no caller could ever see it succeed.
  It also never stamped `last_booster_claim`, so the "once every 30 days"
  cooldown could never actually engage. Both are fixed: a successful
  redemption now returns `204` and updates the claim timestamp.
- `tickets.user_id` had no foreign key constraint to `users(user_id)` at
  all, just a plain column — so the account data-export/deletion pipeline
  (`POST /users/{id}/data`, `routes/users/endpoints/create_data_task`)
  silently skipped every ticket a user had ever filed. The walker
  (`ddr_task.go`) auto-includes any table with a real FK into an
  already-registered root (`users`/`teams`), so the fix is schema-only:
  `exp/ticketuserfkey.sql` adds the constraint `NOT VALID` (4 legacy
  tickets reference since-deleted accounts; `NOT VALID` enforces it for
  all new/updated rows without deleting or nulling that history). No Go
  changes needed — confirmed via a direct `pg_constraint` check against
  the dev DB that tickets are now walked correctly.

## [1.2.0] - 2026-08-14

### Added

- Packs are no longer bots-only: a new `pack_type` column (`bot` | `server`
  | `emoji`, immutable after creation) generalizes the existing `BotPack`
  type, and a new `pack_emojis` table backs a genuinely new capability —
  user-curated emoji packs, each emoji its own durably-uploaded asset (not
  a live reference into a server's synced emoji list, so a pack keeps
  working even if the source server stops syncing or leaves). Server packs
  reuse the `Servers []string` field that already existed on `BotPack` but
  was never wired to any route or UI. `add_pack`/`patch_pack` validate
  content per type (bot packs need `bots`, server packs need `servers`,
  emoji packs need `emojis`, capped at 50), `get_all_packs` gained an
  optional `?pack_type=` filter, and a new `edit_packs` entity permission
  (`teams.GetEntityPerms`'s new `"pack"` case, single-owner only — no team
  fallback) lets the existing generic upload-permission-check flow cover
  pack emoji uploads the same way it already covers bot/server banners.
- A generic content-report system (`popplio/reports`, new `routes/reports`
  package), built alongside the pack generalization above to give users a
  way to flag a pack (or, later, any votable entity) for e.g. a license
  violation on an emoji pack. `PUT /users/{uid}/{target_type}/{target_id}/reports`
  mirrors the votes router's exact URL shape and target-type handling.
  Reports are keyed `(target_type, target_id)`, same convention as
  `entity_votes`; a partial unique index allows only one open report per
  reporter per target, and a per-user daily cap (10) limits spamming many
  different targets. Reporter identity is never exposed outside the staff
  panel — the public API never returns it. Reviewed exclusively through a
  new Arcadia RPC (`UpdateReports`/`ReportAction`, following
  `PartnerAction`'s exact discriminated-union codec pattern) gated on a new
  `review_reports` staff permission; there is deliberately no public
  listing/review route, matching how Blog/Partners never got one either.
  **Config/DB note:** three new one-off migrations to apply —
  `exp/packtype.sql`, `exp/packemojis.sql`, `exp/reports.sql`.
- `GET /bots/@all` and `GET /servers/@all` gained an optional
  `?sort=trending` param, ranking by net votes (upvotes minus downvotes) in
  the last 7 days instead of newest-first, and returning only entities with
  at least one vote in that window. New composite index
  `entity_votes_target_created_idx` (`exp/entityvotesidx.sql`) backs the
  underlying grouped query — `entity_votes` had no index at all before
  this, so trending would otherwise have been a full table scan.
- `GET /reports/stats`: a new, deliberate exception to the reports
  system's "no public read-back" design — anonymized counts of reports
  grouped by `reason`/`status` only (no report IDs, no target identity, no
  reporter identity), for a public moderation-transparency page.
- `GET /servers/@emojis`: a new paginated endpoint returning only
  `server_id`/`name`/`avatar`/`emojis`/`stickers` for servers with
  `show_emojis = true`. `IndexServer` (what `@all` returns) excludes
  emoji/sticker data entirely, so a cross-server emoji/sticker browse page
  had no way to bulk-fetch this without N+1 calls to `GET /servers/{id}`
  before this.

## [1.1.0] - 2026-08-13

### Added

- Infernoplex the standalone Rust Discord server-tracking bot has been
  ported into Popplio's own binary as a new `infernoplex/` package, the same
  treatment Arcadia got earlier. `main.go` now starts it right alongside
  Arcadia (`infernoplex.Start(state.Context)`, stopped with the same 30s
  grace period on shutdown) instead of it running as a separate service.
  The port covers everything the Rust bot did: a guided multi-step server
  setup wizard (`infernoplex/bot/setup.go`), invite creation/resolution
  (`infernoplex/invite`), a server-info push command gated on "Edit
  Servers" (`cmdUpdate`), a vote leaderboard (`cmdLeaderboard`), a
  bot-stats command (`cmdStats` version/Go version/git commit/env, mirrors
  Arcadia's `/info`), and background tasks for server/emoji/sticker sync and
  team-member cleanup (`infernoplex/tasks`). It also runs its own small
  internal HTTP API, "Sorbet" (`infernoplex/sorbet`), structured the same
  way as Arcadia's panel dispatch. The standalone Rust Infernoplex service
  is superseded by this and should be decommissioned.
  **Config shape change (update `config.yaml` before deploying):** a new
  `infernoplex:` block with `client_id`/`client_secret` plus per-environment
  `prefix`/`server_port`/`token` (same `Differs[T]` staging/prod/beta/dev
  pattern used elsewhere) — see `config.yaml.sample`.
- Infernoplex's leaderboard command now replies with a "No Votes Yet" embed
  instead of an empty/broken one when a server has zero votes.
- Self-hosted proof-of-work vote captcha (`popplio/captcha`), replacing the
  dead `HCaptchaInfo` scaffolding in `types/vote.go` (which was never wired
  to anything) with something actually enforced. `GET
  /votes/captcha/challenge` issues a signed, stateless hashcash-style
  challenge (find a nonce so `sha256(salt+":"+nonce)` has N leading zero
  bits); `PUT .../votes` now requires a solved challenge in the request body
  for bot/server votes unless the entity has opted out via the existing
  `captcha_opt_out` setting. Challenges are HMAC-signed with the new
  `captcha.hmac_secret` config value so they can't be forged, and each
  solved challenge is single-use (consumed in Redis on first successful
  verification) so a solve can't be replayed across multiple votes. No
  third-party captcha provider involved — the whole protocol lives in
  `popplio/captcha`.
  **Config shape change (update `config.yaml` before deploying):** a new
  `captcha:` block with a per-environment `hmac_secret` (same `Differs[T]`
  pattern used elsewhere) — see `config.yaml.sample`. Generate one with e.g.
  `openssl rand -hex 32`; rotating it invalidates all outstanding
  challenges.

### Changed

- Rebranded "Infinity List" → "Omniplex" across every remaining user-facing
  string that still had the old name: the MFA issuer shown in a staff
  member's authenticator app on re-enrollment (`arcadia/panel/mfa.go`), the
  staff bot's `/analytics` embed title (and its frozen conformance string),
  and the fallback SEO description on `GET .../teams/{id}/seo` when a team
  has no custom short description (`"View the team X on Omniplex"`). A few
  doc comments got the same treatment with no functional effect.
- Every reply the staff bot makes is now an embed, including one-liners
  (the "Isabelle" rewrite, #43). A bare content message is indistinguishable
  from a staff member talking, which matters in the staff server where the
  bot's answers and the conversation share a channel. `Ctx.Say` builds the
  embed itself, so this is a change of container rather than of wording —
  every string frozen in `arcadia/conformance` is untouched and still
  asserted. Two coloured variants went in alongside it: `Ctx.Fail` (red) for
  the command guards, the panic handler and the "there was an error" paths,
  and `Ctx.Ok` (green) for the 16 replies that report something having
  worked, so a refusal is visibly different from an answer without either
  having to say so. The modal driver and the permission editor's ephemeral
  refusals, which answer through the interaction rather than through `Ctx`,
  build the same shape by hand (`modalReply` in
  `arcadia/bot/interactions.go`). `TestRepliesAreEmbeds` walks the package's
  AST for any `MessageCreate` that sets `Content` and fails if one appears.
- A second pass over the same files, this time pulling out the repetition
  rather than only moving it. In `arcadia/rpc`: `modLogReason` builds the
  mod-log embed nine handlers were each building by hand (title,
  description, one Reason field, footer, colour), `reasonField` covers the
  four multi-field embeds that keep their own shape, and
  `guardBot`/`guardUser` replace the ten copies of "reject an over-long
  reason, then check the target exists". `certifyAdd` went from 47 lines to
  20 this way, and `review.go` split into `claim.go` and `verdict.go` once
  it had. In `arcadia/panel`: `authorize` replaces the ten copies of the
  twelve-line `checkAuth` + `resolvedPerms` preamble, and `ops_core.go`
  (608) split into `ops_auth`, `ops_hello`, `ops_queue`, `ops_rpc`,
  `ops_search` and `ops_proxy`. `arcadia/tasks/staffresync.go` (579) split
  into the resync itself, its reporting and its Discord role mirroring.
  What was deliberately *not* factored out: the frozen embed and error
  strings stay written out at their call sites, because
  `arcadia/conformance` finds them by scanning the source for the literal —
  a helper that formatted them would pass its own tests while quietly
  removing that check. For the same reason the SQL stays literal at each
  call site, since `arcadia/dbconform` PREPAREs every string literal it can
  find against a real database. And the five steps of `StaffResync` are
  left inline: they share a transaction and a working set that each step
  narrows, so splitting them would make an ordering that is load-bearing
  look optional.
- The five files that had grown past the point of being navigable are split
  by what they do, with no behaviour change: `arcadia/bot/staffroles.go`
  (1137) into `staffmgmt.go` (the role model, the authority rules, the
  lookups), `staffroles.go`, `staffperms.go` and `staffrender.go`;
  `arcadia/bot/commands.go` (697) into `commands.go` (the registry and the
  two shared RPC helpers) plus `help.go`, `invites.go`, `stats.go` and
  `staffops.go`; `arcadia/panel/ops_shop.go` (861) into one file per shop
  concern (tiers, items, benefits, coupons, whitelist);
  `arcadia/panel/ops_staff.go` (759) into positions, members and
  disciplinaries, with its two shared existence checks moved to
  `ops_query.go` where the shop operations that also use them can find
  them; and `arcadia/bot/permeditor.go` (858) into
  session/apply/render/util. Each new file opens with what it covers and
  what is non-obvious about that area. All of it is code movement verified
  line-for-line against the original; nothing in the repo's directory
  structure changed, and `routes/`'s one-package-per-endpoint layout is
  left alone since `uapi` requires it.
- `arcadia/rpc/methods.go` (878 lines, every RPC action in one file) is
  split into one file per group of actions, grouped exactly the way
  `types.rpcPermissions` groups them — so the file an action lives in is
  the same question as which permission gates it: `review.go` (claim,
  unclaim, approve, deny, unverify), `certify.go`, `transfer.go`,
  `forceremove.go`, `premium.go`, `votes.go`, `apps.go`, plus `dispatch.go`
  for the method-to-handler switch and `audit.go` for the
  `staff_general_logs` write. `core.go` keeps the `Execute` pipeline and
  the shared guards, and its package doc now carries the map of where
  things live and the note that every mod-log embed is reproduced verbatim
  from the Rust original (that note used to sit above the dispatcher and
  said "every embed below", which the split would have made a lie). Pure
  code movement: every moved line is byte-identical to what it replaced,
  and `arcadia/conformance` scans the whole package rather than one file,
  so it pins the embed strings exactly as before. (`review.go` was later
  split further into `claim.go`/`verdict.go` — see the dedup-pass bullet
  above; `arcadia/CONFORMANCE.md`'s file references still say `review.go`
  in a few spots and need updating to match, see Known Issues below.)
- The Dev Team staff application no longer requires or mentions Rust
  (description and two questions updated to reflect Go/TypeScript only);
  the QAQC application track was removed entirely. Consistent with Arcadia
  and now Infernoplex both being fully off Rust.

### Security

- Only the prod instance now sets the main Discord bot's gateway presence
  (`state.go`'s `OnGuildsReady` handler). Staging/beta/dev instances still
  connect and function normally, they just no longer call
  `SetPresenceForShard`, so a non-prod checkout — misconfigured shared
  token or otherwise — can never overwrite what the public bot's profile
  shows as its "Watching" activity.

### Removed

- Five retired permissions — `view_shop`, `manage_shop`,
  `manage_bot_whitelist`, `view_cdn`, `manage_cdn` purged from every
  stored permission array (`staff_positions.perms`,
  `staff_members.perm_overrides`, `staff_disciplinary_types.perm_limits`)
  via a new one-off migration, `exp/rewrite/remove_broken_perms.sql`
  (needs to be applied manually against the database like other `exp/`
  scripts).

### Known issues found during this pass, not yet fixed

- Infernoplex's new "No Votes Yet" message has a typo: "Unfortuently, your
  server has no votes at this time."
- `config/config.go`'s `Naevis` struct (added alongside `Infernoplex` as an
  apparent placeholder for a second bot) is dead code it's never
  referenced from the top-level `Config` struct despite its fields being
  tagged `validate:"required"`, and `config.yaml.sample`'s `naevis:` section
  was already removed. Safe to delete outright, or finish wiring it up if
  Naevis is still planned.
- `arcadia/CONFORMANCE.md` references a `arcadia/rpc/review.go` in a few
  places (issues #6, #9, #10) that doesn't exist the file is `verdict.go`.
  Looks like a stale rename from drafting the Isabelle split.

## [1.0.1] - 2026-08-05

### Changed

- Every `Differs[T]` config key (DB tokens, site URLs, etc.) previously
  required *both* a `staging` and a `prod` value to be set regardless of
  which environment a given box actually runs — a `current-env: prod` box
  was rejected at startup for a missing `staging` value it would never
  read, and vice versa. `ValidateDiffers` now only requires whichever value
  `Parse()` will actually resolve for `CurrentEnv` (`prod` needs `prod`,
  `staging` needs `staging`; `beta`/`dev` still accept either their own
  value or a `staging` fallback, unchanged), so a config file only needs to
  fill in what the box it's deployed to actually uses.
- Staff roles and permissions can now be managed interactively from the staff
  bot: `/staffroles edit [role]` and `/staffperms edit <user>` open a select
  menu editor (`arcadia/bot/permeditor.go`) with a role picker, a category
  picker and a multi-select of that category's permissions, preselected to
  what the role or member currently holds — ticking grants, unticking revokes,
  and everything outside the open category is carried through untouched.
  Dangerous permissions are marked ⚠️ and ones the caller cannot manage 🔒.
  The existing one-at-a-time `grant`/`revoke` subcommands are unchanged and
  still work; both paths share the same rank check and `perms.CheckPatch`
  rule, and the editor re-checks both at the moment of the write rather than
  only when it opened, since a session lives for ten minutes. Every render
  reloads the target from the database, so two people editing the same role
  see each other's changes instead of saving a stale picture over them.
  Alongside the menus are buttons: "Grant all"/"Revoke all" for the open
  category (which leave permissions the caller cannot manage exactly as they
  are, so one locked permission doesn't make the button useless), "Pick
  another role" to switch targets without closing, and "Close". `edit` is
  registered as the *first* subcommand of both commands, since Discord lists
  them in registration order and never lets the parent command
  (`/staffroles` on its own) be invoked at all.

### Fixed

- `DELETE /teams/{tid}/members/{mid}` had its last-owner safety check
  inverted (introduced by the kittycat→internal `perms` package refactor):
  it fired when the member being removed was *not* an owner instead of when
  they were, so removing any regular member from a team with only one owner
  (the common case) 400'd with "There needs to be one other global owner
  before you can remove yourself from owner" — while actually removing the
  team's last real owner sailed through with no check at all, the exact
  case this was meant to prevent. Condition un-inverted.
- Every staff bot slash command appeared twice in every server. The bot
  registers its commands per guild (`arcadia/bot.SyncCommands`), but the
  application still carried global registrations of the same commands from an
  earlier deployment, and Discord lists a global command alongside a guild
  command of the same name rather than letting the guild copy take its place.
  `SyncCommands` now finishes by deleting the global registration of any
  command it registers per guild (`pruneGlobalCommands`), so the duplicates
  clear themselves on the next sync (startup, or `/register`). Global
  commands whose names the bot does not register are left alone and only
  logged as a warning, since they belong to something else sharing the
  application.
- The `server`/`team` auth types were never registered as OpenAPI security
  schemes (only `User`/`Bot` were, via `docs.AddSecuritySchema` in
  `main.go`) even though `Authorize()` has always fully supported them —
  every one of the 41 operations requiring `server` or `team` auth
  (`PUT /bots`, `PUT /servers`, both `PATCH .../settings` endpoints,
  reviews, sessions, etc.) referenced a security scheme name absent from
  `components.securitySchemes`. Harmless to the API itself, but any tool
  that resolves the requirement against registered schemes crashes outright
  on the unresolved reference — including the docs site's OpenAPI reference
  pages (`fumadocs-openapi`'s `APIPage`, which throws
  `Cannot read properties of undefined (reading 'type')`). Registered both
  (`docs.AddSecuritySchema("server", ...)` / `("team", ...)`, lowercase to
  match `AuthTypeMap`'s self-mapping for these two types).
- Presence still never actually got set even after 1.0.0's fix, now logging
  `error while setting presence err="no gateway configured"` from inside
  `OnGuildsReady` instead of right on startup — that fix only addressed the
  timing, not the actual cause: Popplio runs sharded (`OpenShardManager`),
  and `Discord.SetPresence` only ever checks disgo's single-gateway field
  (populated by `OpenGateway`, not `OpenShardManager`), so it returns
  `ErrNoGateway` unconditionally on a sharded bot regardless of readiness.
  `OnGuildsReady` also fires once per shard, not once globally. Now uses
  `Discord.SetPresenceForShard(ctx, event.ShardID(), ...)` instead.
- `POST /auth/test` ("Test Auth") 500ed on every call that reached an actual
  authorization check — `api.Authorize` reads `PERMISSION_CHECK_KEY` out of
  the route's `ExtData` unconditionally, but the synthetic `uapi.Route{}`
  this endpoint builds to call it never set `ExtData` at all, so any request
  with a syntactically valid token failed with a 500
  (`permissionCheck not found in route.ExtData`) instead of returning
  whether the token is actually valid. Only requests with a token that
  failed even earlier (nonexistent in `api_sessions`) ever got a real
  response (401). Now sets a no-op `PermissionCheck` (`NeededPermission`
  always returns `nil`), since this endpoint has no permission model of its
  own to enforce — it's purely "is this token valid for this target."

### Removed

- The `use_borealis` staff permission. Borealis was removed from the platform
  during the port (`arcadia/CONFORMANCE.md` D11a — the `arcadia.borealis_url`
  config key, the client and the `Approve` call to it are all long gone), so
  the permission has gated nothing since and only added a line to
  `/permissions` and a row to every permission picker. `exp/rewrite/flatperms.sql`
  now lists the old `borealis.*` in `retired_perm` (dropped on purpose)
  instead of mapping it onto `use_borealis`, and
  `exp/rewrite/remove_borealis_perm.sql` strips it from
  `staff_positions.perms`, `staff_members.perm_overrides` and
  `staff_disciplinary_types.perm_limits` for databases the old migration
  already ran against. That cleanup is needed rather than cosmetic: the
  permission model deliberately keeps names it does not declare, since they
  may belong to another service, so `use_borealis` would otherwise sit in
  those columns for good and show up under "Other services".

### Security

- Bot accounts can no longer hold staff permissions at all — not through a
  staff role, not through a direct grant, and not through `arcadia.owners`
  (`perms.ErrBotAccount`). Previously nothing stopped one: `StaffResync`
  walks every member of the staff server and creates a `staff_members` row
  for anyone holding a position's Discord role, and it never looked at
  whether that member was a bot, so giving a bot a staff role in Discord
  handed it that role's permissions — including through the panel session
  and RPC paths, which only ever asked what the row said. A bot is a token
  that can be handed to another program, which is exactly what the staff
  model's accountability assumes cannot happen, and nothing needs it: the
  staff bot and the panel both act under a staff member's identity, never
  their own. Enforced on both sides:
  - Reads: `perms.StaffGrants` carries a `BotAccount` flag, joined in from
    dovewing's user cache by `LoadStaff` at no extra cost, and `Resolve()`
    returns nothing and `Rank()` returns `NoRank` when it is set. The panel's
    session check (`impls.CheckAuthInsecure`), its login
    (`ops_authorize.go`) and its member view (`impls.GetStaffMember`, whose
    additory disciplinaries could otherwise add permissions on top of an
    empty set) all apply the same rule. These paths stay database-only, so
    they keep working when Discord does not.
  - Writes: `perms.RejectBotAccount` resolves through dovewing all the way
    to Discord if the account has never been seen, and fails closed if it
    cannot tell. `StaffResync` now skips bot members entirely, which also
    means an existing bot's staff row is cleaned up by the same pass that
    handles members who left; the panel's `editMember` and the staff bot's
    `/staffperms grant`/`revoke`/`edit` refuse a bot target outright.

## [1.0.0] - 2026-08-04

### Added

- `current-env` now also accepts `beta`, a fourth environment alongside
  `staging`/`prod`/`dev`. Every `Differs[T]` config key gains an optional
  `beta` value (`config.Differs[T].Beta`), consulted only when `current-env`
  is `beta` and falling back to `staging` when unset — same mechanism as
  `dev`'s override, but without `dev`'s relaxed Staging/Prod requirement:
  `beta` is validated exactly like `staging`/`prod` (`ValidateDiffers`),
  since it's a real running deployment rather than a personal machine. In
  practice this means most config (DB, tokens, etc.) can stay shared with
  staging, and only keys that genuinely differ per deployment — like
  `sites.frontend` — need an explicit `beta:` value.
- `bgtasks` package: a new home for Popplio's own periodic background jobs,
  separate from `arcadia/tasks` (the staff bot's jobs, which only run when
  Arcadia is configured) so core platform features don't depend on staff
  tooling being set up. First job: `bot_uptime_check`, which periodically
  records whether every listed bot is currently online in the main server
  into `bots.uptime`/`total_uptime`/`uptime_last_checked`. These columns
  have existed since the Rust port but were never actually written to —
  Arcadia's old uptime checker (`src/tasks/__toberewritten/uptime.rs`)
  didn't even compile against the serenity version it was last touched
  against, and was explicitly never ported (see `arcadia/CONFORMANCE.md`).
  Reads presence straight from Popplio's own gateway cache (it already
  requests the Presence intent) rather than Infernoplex, which deliberately
  never requests it.
- `servers.avatar`: servers previously had no icon anywhere (index listing,
  detail page, or the staff panel's server search all showed a blank/
  initials fallback) — the old cache-server subsystem used to synthesize
  this from its own CDN cache, and nothing replaced it after that was
  retired (`exp/remove_cache_servers.sql`). Populated once at Add Server
  time from the invite resolution already done there, and kept fresh
  afterward by Infernoplex's `serversync` task, which now also syncs every
  listed server's icon (not just opted-in ones' emojis/stickers) from its
  gateway cache. Requires the new `servers.avatar` column
  (`exp/add_servers_avatar.sql`, needs to be applied manually against the
  database like other `exp/` scripts).
- Webhooks gained a new `hmac_auth` mode (`hmac_auth` on
  `POST`/`PATCH .../webhooks`): the payload is sent as plain JSON with an
  `X-Webhook-Signature: sha256=<hex hmac>` header, the same shape GitHub and
  Stripe webhooks already use. It's now the recommended mode for new
  webhooks the previous default ("splashtail": AES-GCM encrypted body,
  nonce-chained double HMAC across two headers) required implementing
  decryption just to verify a delivery, not just a signature check.
  Existing webhooks are unaffected: `hmac_auth` defaults to off and the
  splashtail/`simple_auth` protocols are unchanged and fully supported 
  this only adds a third option, it doesn't remove or alter the other two.
  Requires the new `webhooks.hmac_auth` column
  (`exp/webhookhmacauth.sql`, needs to be applied manually against the
  database like other `exp/` scripts).
- `current-env` now also accepts `dev`, a third environment alongside
  `staging`/`prod`. Every `Differs[T]` config key (`config/config.go`) gains
  an optional `dev` value, only consulted when `current-env` is `dev`, and
  only used if actually set — an unset `dev` value falls back to `staging`,
  so no existing `config.yaml` needs to change. Lets a local checkout run
  against things like a personal Discord bot application
  (`discord_auth.token`, `arcadia.token`) without touching the real staging
  config. `discord_auth.token` (Popplio's own bot token) is now itself a
  `Differs[string]` rather than a single flat value, so it can differ across
  environments the same way Arcadia's staff bot token already could.
  Anything gated to "real production" (Paypal live vs sandbox API base,
  Arcadia's background tasks, the staff bot's guild-member-join
  announcements, the staging-sensitive-permission gate) now treats `dev` the
  same as `staging` rather than falling through to production behavior.

- `PUT /servers` add a server to the list directly from a Discord invite
  link. Resolves the guild via the invite (the tracking bot does not need to
  already be in the server), rejects duplicates and blacklisted vanities,
  and auto-creates an owning team the same way bot submission already does
  (or attaches to an existing team the submitter has `bot.add`-equivalent
  permission on).
- Packs can now include servers alongside bots: a `servers` column,
  resolution into full `IndexServer` objects, and matching validation on
  both `POST /packs` (create) and `PATCH /packs/{url}` (edit). A pack must
  contain at least one bot or server between the two fields.
- Bots can self-report presence (`online`/`idle`/`dnd`/`offline`) via
  `POST /bots/stats`, alongside the existing server/shard/user stats. The
  reported value is folded into the resolved `user.status` returned
  everywhere a bot's info appears, since most bots don't share a guild with
  the tracking bot for a real gateway presence to be read from.
- Bots with no explicit self-reported status but a real track record of
  posting stats (a nonzero server count from a stats post within the last
  24 hours) are now shown as `online` rather than falling back to
  dovewing's almost-always-offline gateway-derived status.
- `GET /servers/meta?invite=...` resolves a Discord invite to a preview of
  the server it points to (name, icon, member counts, and whether it's
  already listed) without adding anything — lets a client show what's about
  to be submitted before Add Server is actually called. Shares its invite
  resolution logic with `PUT /servers` via a new `ResolveInvite` helper.
- Servers can opt in to showing their custom emojis and stickers on their
  listing page via a new `show_emojis` setting (`PATCH /servers/{id}/settings`).
  `GET /servers/{id}` now includes `emojis`/`stickers`/`emojis_synced_at`,
  always empty unless the owner has opted in. The actual snapshot is synced
  periodically by the tracking bot (Infernoplex), not fetched live per
  request, and requires the bot to currently be a member of the server —
  Popplio itself never talks to Discord for this.
- `GET /servers/meta` now also reports `bot_present`/`bot_invite_url` by
  asking Infernoplex's Sorbet API whether the tracking bot is currently a
  member of the resolved guild, via a new `CheckBotGuildPresence` helper.
  Best-effort: any failure to reach Infernoplex is treated as "not present"
  rather than failing the request.

### Changed

- Bots now support downvotes, matching servers/teams/packs
  (`votes.EntityVoteInfo` no longer hardcodes `SupportsDownvotes = false` for
  the `bot` target type).
- `meta.popplio_proxy` now defaults to `https://gateway.nodebyte.host/proxy/discord`
  (the shared parent-company gateway), replacing the old local
  `http://127.0.0.1:3219` twilight-http-proxy convention. Both Popplio's own
  bot client (`state.Setup`) and Arcadia's separate staff bot
  (`arcadia/dclient`) now route their REST traffic through it via
  `rest.WithURL`/`rest.WithHTTPClient` (`state.ProxyRestOpts`). Since that
  gateway authenticates every request with its own shared bot credential by
  default, each client sends its own token via an `X-Upstream-Authorization`
  header instead, which the gateway forwards as the real `Authorization`
  header sent to Discord — so Popplio and Arcadia's staff bot each keep
  their own distinct bot identity rather than both authenticating as
  whichever bot the gateway holds.
- `EntityGetVoteCount` (used by nearly every bot/server/team/user/pack
  detail and list endpoint) now counts up- and down-votes in a single query
  with `FILTER`, instead of two separate `COUNT(*)` round trips.
- Bot/server index resolution (`ResolveIndexBot`/`ResolveIndexServer`,
  called by `GET /bots/@all`, `GET /servers/@all`, search, random, the bots
  index, packs, team entities, and user profiles) now resolves every row in
  a page concurrently via `errgroup` instead of one row at a time — each
  row's dovewing/vanity/vote lookups are independent, so a page of results
  no longer pays for them sequentially.
- `GET /list/current-status` now issues both the Instatus and UptimeRobot
  requests with the request's own context and a bounded client timeout,
  instead of an unbounded `http.Get`/`http.NewRequest` that could hang the
  handler indefinitely if the upstream stalled.
- Deduplicated the `page` query-parameter parsing copy-pasted across nine
  endpoints (each with a slightly different error response for the same
  invalid-page case) into a shared `pagination.Parse` helper.
- `DELETE /users/{uid}/packs/{id}` and `PATCH /users/{uid}/packs/{id}` each
  folded two sequential "does the pack exist" / "who owns it" queries into
  one.
- The generic error bodies returned when a failure carries no specific
  message of its own (`constants/constants.go` — 404s, 400s, 403s, 401s,
  500s, 405s, and missing-body errors) were all a "Slow down, bucko!" joke
  string. Replaced with plain, professional messages that actually describe
  the failure.

### Fixed

- The `server`/`team` auth types were never registered as OpenAPI security
  schemes (only `User`/`Bot` were, via `docs.AddSecuritySchema` in
  `main.go`) even though `Authorize()` has always fully supported them —
  every one of the 41 operations requiring `server` or `team` auth
  (`PUT /bots`, `PUT /servers`, both `PATCH .../settings` endpoints,
  reviews, sessions, etc.) referenced a security scheme name absent from
  `components.securitySchemes`. Harmless to the API itself, but any tool
  that resolves the requirement against registered schemes crashes outright
  on the unresolved reference — including the docs site's OpenAPI reference
  pages (`fumadocs-openapi`'s `APIPage`, which throws
  `Cannot read properties of undefined (reading 'type')`). Registered both
  (`docs.AddSecuritySchema("server", ...)` / `("team", ...)`, lowercase to
  match `AuthTypeMap`'s self-mapping for these two types).
- `PUT /servers` and `PUT /bots` still wrote the legacy wildcard string
  `global.*` into a new team's `team_members.flags` when creating the
  owner's membership, instead of the flat model's `owner` permission
  (`perms.EntityOwner`). `exp/rewrite/flatperms.sql` converts this
  correctly for existing rows, but every server/bot added *after* running
  that migration created a team whose owner held a permission string the
  flat permission checker doesn't recognize as anything — silently locking
  them out of managing their own new listing (`edit_servers`/`edit_bots`
  checks fail, since `global.*` isn't `owner` and isn't a declared
  permission either). `arcadia/tasks/cleaners.go`'s `TeamCleaner` task had
  the same bug in both directions: it looked for orphaned-of-owner teams by
  querying `flags @> ARRAY['global.*']` (which will now never match
  anything, since the migration already converted every existing row) and
  wrote `global.*` back when promoting a replacement owner. All three now
  use `perms.EntityOwner`.
- Every mod-log notification embed (`PUT /servers`, `PUT /bots`,
  `DELETE /bots/{id}`, and the two `PATCH .../settings` endpoints below)
  built its link back to the site with `Sites.Frontend.Production()`,
  forcing the production URL regardless of which environment the action
  actually happened in. On staging/beta, this meant either a broken link
  (if `sites.frontend.prod` wasn't configured on that box at all — its
  `Production()` has no fallback, so an unset value silently becomes an
  invalid relative-path embed URL and Discord rejects the whole message
  with `50035`) or a link to an entity that only exists in a different
  environment's database. Switched to `.Parse()`, which resolves against
  whichever environment is actually running.
- `PATCH /servers/{id}/settings` and `PATCH /bots/{id}/settings` returned a
  500 whenever their mod-log notification embed failed to send — including
  the guaranteed case for servers, which built its embed with
  `Thumbnail: &discord.EmbedResource{}` (a present-but-empty resource,
  which Discord's API rejects outright with `50035: Invalid Form Body`
  rather than treating as "no thumbnail"). The underlying update had
  already succeeded in both cases — the error message even said so — so a
  caller retrying on this 500 risked double-submitting. The thumbnail is
  now omitted when there's no avatar instead of sent empty, servers' embed
  now uses the real `servers.avatar` value instead of nothing, and a
  failure to post the notification is logged rather than failing the
  request, matching the existing pattern in `PUT /servers`.
- `PUT /servers` wrote every field into the wrong column: `createServerArgs`'s
  hand-written value order didn't match `types.CreateServer`'s field
  declaration order, which is what `db.GetCols`/the generated column list
  actually follow. Values were bound to columns purely by position, so e.g.
  `server_id` was written into `invite`, `name` into `short`, and
  `extra_links` (a `[]Link`) into `tags` (a `text[]`) — the last of which is
  what surfaced as a `cannot find encode plan` error, since a `[]Link` can't
  encode into a `text[]` column. `createServerArgs` now lists values in the
  same order as the struct, with a comment on both explaining they must stay
  in sync (the existing length check on `createServerColsArr`/`serverArgs`
  only ever caught the two lists having different lengths, not entries being
  out of order relative to each other).
- `servers.extra_links` was still `text[]` in the deployed database while
  the application code (and `data/seed-ci.json`'s own schema conformance
  check) has expected `jsonb` for a while, matching `bots`/`teams`/`users`'
  `extra_links` columns. Migrated via `exp/fix_servers_extra_links_type.sql`
  (needs to be applied manually against the database like other `exp/`
  scripts, same as `exp/rewrite/*.sql`). Independent of the column-ordering
  bug above, but was masking it: the ordering bug meant `extra_links`
  received whatever value happened to land at that position, which could
  vary by field type in ways that made this fix look sufficient on its own.
- Presence never actually got set, always logging
  `error while setting presence err="no gateway configured"`. First traced to
  `Discord.SetPresence` being called right after `Discord.OpenShardManager`
  returned (before the shards finish their handshake) and moved into the
  `OnGuildsReady` handler — which turned out to only fix the timing, not the
  actual cause: Popplio runs sharded (`OpenShardManager`), and
  `Discord.SetPresence` only ever checks disgo's single-gateway field
  (populated by `OpenGateway`, not `OpenShardManager`), so it returns
  `ErrNoGateway` unconditionally on a sharded bot regardless of readiness.
  `OnGuildsReady` also fires once per shard, not once globally. Now uses
  `Discord.SetPresenceForShard(ctx, event.ShardID(), ...)` instead.
- `current-env: dev` still required a real `staging` and `prod` value for
  every `Differs[T]` config key, and for every Arcadia staff-server
  channel/role/server ID, defeating the point of `dev`: a local checkout
  needed a fully populated staging/prod config, including Arcadia secrets,
  just to start. `Differs[T]`'s requirement is now environment-aware (`dev`
  only needs `dev` or a `staging` fallback to resolve; `staging`/`prod`
  keep the original both-required behavior), and Arcadia's staff-server
  fields use a new `requirednotdev` validator so they're only required
  outside of `dev`.
- `GET /servers/meta`'s route registration was missing the `ExtData` entry
  `BaseSanityCheck` requires whenever a route declares `Auth`, so the
  server panicked at startup ("Base sanity check failed: permissionCheck
  not found in route.ExtData") before it could serve a single request.
- `PATCH .../webhooks/{id}` silently ignored `simple_auth` in the request
  body — the `UPDATE` statement never included that column, so a webhook's
  auth mode could only ever be set at creation, never changed afterward.
- `GET /list/current-status`'s Redis cache never actually worked: it passed
  a raw `map[string]any` to `Set`, which go-redis cannot serialize (returns
  `"redis: can't marshal map[string]interface{}"`) — an error that was
  never checked, so every request silently round-tripped to Instatus or
  UptimeRobot instead of using the 3-minute cache the code's own comment
  says exists. Now JSON-marshals before `Set` and unmarshals back into the
  same shape on a hit, so the response is identical whether it came from
  cache or not (previously a hit, had it ever occurred, would have returned
  a raw string instead of the status object a miss returns).
- `add_review`/`edit_review`/`remove_review` each called
  `state.Redis.Del(ctx, "rv-"+targetId+"-"+targetType)` on every mutation,
  invalidating a cache key that is never `Set` or `Get` anywhere in the
  codebase — three no-op Redis round trips per review change. Removed.
- The OAuth authorization-code replay check (`create_oauth2_login`) used a
  separate `Exists` followed by `Set` to mark a code used, which is not
  atomic: two concurrent requests carrying the same code could both pass
  `Exists` before either called `Set`, letting the same code be redeemed
  twice — exactly the race the code's own comment says it closes, but
  didn't. Now uses `SetNX`, which checks and marks the code used in one
  atomic round trip.
- Several route handlers (`delete_pack`, `patch_pack`, `current_status`)
  returned a bare `uapi.DefaultResponse(http.StatusInternalServerError)` on
  DB/upstream failures without logging anything, so some production 500s
  were invisible in the logs. They now go through `resp.Err`, which is what
  the shared `api/resp` package exists for.
- A background goroutine filtering empty entries out of the Stripe webhook
  IP allowlist mutated the slice while ranging over its original indices, a
  classic Go bug that silently skips the element shifted into a just-removed
  slot — consecutive empty lines in Stripe's IP list could leave stale
  entries in a security-relevant allowlist. Now builds a filtered copy
  instead of mutating in place.
- Startup panicked the entire process on a transient Stripe API/network
  failure (deleting existing webhooks, creating the new one, or fetching its
  IP allowlist) instead of disabling Stripe webhook support and continuing,
  unlike the equivalent Paypal setup a few lines above it, which already
  degraded gracefully.
- `webhooks/sender` used `panic()` as its input-validation strategy for a
  handful of preconditions, including from inside an unrecovered goroutine
  (the randomized "send a bad webhook to test auth" path) — a single
  malformed webhook payload reaching that path could crash the whole
  process rather than just fail one webhook delivery. Preconditions now
  return errors instead.
- Several fire-and-forget goroutines spawned from request handlers (bot/server
  detail-page analytics, review garbage collection, Stripe perk delivery,
  vote logging and webhook dispatch) had no `recover()`, so a panic in any of
  them would take down the whole process instead of just that background
  task. Two of them (`create_user_entity_vote`'s webhook-send goroutine and
  the Stripe perk-delivery goroutine) also wrote to an `err` variable shared
  with their enclosing handler, a data race; both now use a goroutine-local
  variable.
- `data/seed-ci.json`, the schema snapshot the `db_fields_check.py` CI test
  checks Go struct `db` tags against, had fallen out of sync with several
  recently added columns (`bots.self_status`, `packs.servers`,
  `servers.show_emojis`/`emojis`/`stickers`/`emojis_synced_at`), breaking the
  test build. Also found and excluded `bots.cache_server_uninvitable`, a
  real DB column with no corresponding Go struct field anywhere in the
  codebase, via the existing `ignore_fields` convention.
- Team votes were never resolved when a team was embedded inside a user's
  profile response (`GET /users/{id}`) — every embedded team silently
  reported 0 votes regardless of its real count.
- `GET /users/{id}` never requested team member data (`team_member`) when
  resolving a user's teams, only `bot` and `server` — so any client relying
  on that response to determine a user's permissions on a team-owned entity
  (e.g. "can I edit this bot?") always saw an empty permission set.
- The tracking bot's Discord presence update was hard-gated to the
  production environment only, and silently swallowed any error from
  actually setting it (the failure check was reading a stale variable from
  an earlier, unrelated call rather than the real result) — presence now
  updates in every environment and logs real failures instead of going
  silent.
- Nil slices (a user's teams, a team's `bots`/`servers`/`members`, etc.) now
  consistently marshal as `[]` instead of `null` across list/user/team
  endpoints, preventing frontend crashes on `.length`/`.map` against a
  response that has no data yet.
- `GET /teams/{id}` had the same nil-slice gap on `tags`/`extra_links`
  directly (as opposed to the application-resolved slices above, which were
  already covered) — a team with no tags or links set crashed the frontend
  team page outright rather than just rendering emptily.
- `webhooks/sender` failed to build at all: an in-progress refactor had
  extracted `Secret`, `webhookSendState`, and several helper methods
  (`resolveTarget`, `buildRequest`, `notify`, `markFailed`, `logFields`)
  into new `request.go`/`sendstate.go` files, but `sender.go` still carried
  the old duplicate declarations and its own inline copies of the same
  logic. `sender.go` now uses the extracted helpers instead of duplicating
  them.

### Security

- Internal error details (raw Go `error.Error()` text) are no longer
  included in several API error responses, including session
  authentication and a number of other endpoints the underlying error is
  still logged server-side in full, but clients now receive a generic
  message instead of internal implementation details.
