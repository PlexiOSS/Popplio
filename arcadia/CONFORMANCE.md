# Arcadia → Go port: conformance notes

Arcadia (Rust: axum + sqlx + serenity/poise, ~12,800 LOC) ported into Popplio.
The wire format was originally frozen against a live SvelteKit staff panel;
the default everywhere was to reproduce upstream behaviour byte-for-byte,
including its bugs. That SvelteKit panel is gone — Omniplex's own `/admin`
is the only consumer of this wire format now, and it's in this same
`plexicore` project — so a number of the reproduced quirks below have since
been fixed rather than left frozen. What's still marked "Reproduced" either
still needs a product decision, or is genuinely load-bearing for something
observable within Omniplex itself.

Layout:

| Package | Contents |
|---|---|
| `arcadia/types` | Wire DTOs and the tagged-union codec (§3) |
| `arcadia/impls` | Auth/session, permissions, entity managers, dovewing adapter, Discord helpers |
| `arcadia/rpc` | The shared action layer: pipeline + all 18 methods (§7) |
| `arcadia/panel` | Custom net/http server, middleware, dispatcher, all 25 operations (§4, §5) |
| `arcadia/cdnpath` | CDN name/path validators and the granular CDN permission check |
| `arcadia/tasks` | The 12 background tasks and their runner (§12) |
| `arcadia/bot` | Discord command framework, commands, events, guards (§11) |
| `arcadia/conformance` | Frozen-string tests |

---

## (a) Upstream bugs and quirks

Policy: **reproduce all**, document each, and offer the fix as a separate opt-in
patch. Four internal-only defects were fixed because they change nothing a panel
user or staff member can observe; each is called out as FIXED below.

### Reproduced

**1. ~~Partner avatar path mismatch (§5.14)~~ — MOOT** — `arcadia/panel/ops_content.go`
Same story as #4: the avatar-path validation and cleanup this described went
with the CDN removal (D11b). `parsePartner` no longer touches an avatar path
at all — nothing constructs or compares the two paths that used to mismatch.

**2. ~~Shop coupon Option validation makes "unlimited" unreachable (§5.23)~~ — FIXED** — `arcadia/panel/ops_shop_coupons.go`
`max_uses`, `reuse_wait_duration` and `expiry` were validated as
`value.unwrap_or(0) <= 0 → reject`, so a `null` became `0` and failed,
making the "unlimited uses"/"never expires" cases the DTO comments describe
unreachable through this endpoint. `validateCoupon` now checks the pointer
directly (`action.MaxUses != nil && *action.MaxUses <= 0`) so `null` is
treated as "no constraint" while a present-but-non-positive value is still
rejected. Covered by `TestShopCouponNullMeansUnconstrained` and
`TestShopCouponNonPositiveMaxUsesIsRejected` in `integration_test.go`.

**3. ~~`topreviewer_sync` runs with `LIMIT 0` (§12)~~ — FIXED** — `arcadia/tasks/discord.go`
The weekly job stripped the top-reviewer role from every main-guild member
and then re-granted it to the top `0` reviewers — i.e. to nobody, forever.
The Discord `refresh` command already ran the same query with `LIMIT 3`.
`TopReviewerSync` now uses the same limit, so the role actually rotates with
review activity instead of staying permanently empty.

**4. ~~Chunk-id retry loop tests the same id ten times (§5.10)~~ — MOOT** — `arcadia/panel/ops_cdn.go`
This referred to a file that no longer exists: the whole CDN subsystem was
removed per D11b below, taking the chunk-upload code (and this bug) with it.
Left here for the record rather than deleted, since D11b is the authoritative
note on what happened to CDN functionality.

**5. ~~`PopplioStaff` sends the request PATH as `X-Forwarded-For` (§5.25)~~ — FIXED** — `arcadia/panel/ops_proxy.go`
Checked: something does read it. `routes/users/endpoints/create_data_task`
parses `X-Forwarded-For` as the requester's IP for a GDPR data task's audit
record, and `popplioStaff`'s path proxying isn't restricted to `/staff/*` —
`safeJoinPopplio` only rejects paths that escape the API base, so a staff
member could reach that endpoint through this proxy and have their real IP
silently replaced with a URL path in that record. The panel's request context
now carries the caller's actual address (`withClientIP` in `server.go`,
read back via `clientIPFromContext`), forwarded as `X-Forwarded-For` instead
of `q.Path`.

**6. ~~`Claim`'s `testbot` branch is dead (§7.2)~~ — FIXED** — `arcadia/rpc/claim.go`
`type != "pending"` was rejected immediately above, so `type == "testbot"`
could never be reached. Removed. (`Unclaim`'s `testbot` check runs *first*,
so that one's live and untouched.)

**7. ~~Disciplinary type `created_at` is the disciplinary's, not the type's (§8.2)~~ — FIXED** — `arcadia/impls/auth.go`
`StaffDisciplinaryType.created_at` was populated from the *disciplinary*
row, so the panel showed a type as having been created when the punishment
was issued rather than when the type itself was defined. Confirmed unused
in Omniplex's admin UI (`staff/disciplinary-types` never renders it), so no
external contract depended on the wrong value. `disciplinaryQuery` now
selects the type's own `created_at` (aliased `type_created_at`) and
`GetStaffDisciplinaries` uses that instead.

**8. ~~"testing" corresponding-server is accepted but ignored (§12.1)~~ — FIXED** — `arcadia/tasks/staffresync_roles.go`
The panel's position validator (§5.17) accepts link names `main`, `testing`
and `staff`, but `collectCorrespondingRoles` only handled `main` and `staff`,
warning-and-skipping anything else. A position configured with a `testing`
corresponding role validated cleanly and then silently never synced. Added
the missing `case "testing": guildID = state.Config.Servers.Testing`.

**9. ~~`Unverify`'s mod-log embed has an empty field name~~ — FIXED** — `arcadia/rpc/verdict.go`
The third embed field was built with an empty `name`, which the Discord API
rejects (field names must be 1–256 characters), so the embed post failed and
`Unverify` reported failure *after* having already flipped the bot to
`pending` (the DB write was never rolled back). The field now has a real
name (`"Bot"`, matching `unverifyServer`'s existing `"Server"` field).

**10. `Approve` posts to Discord inside the transaction (§7.2)** — `arcadia/rpc/verdict.go`
Half of this entry is stale: the Borealis call it originally described is gone
(D11a). What remains is real — the Discord mod-log post still happens before
`COMMIT`, holding the transaction open across one network call. Preserved,
since restructuring changes failure semantics (today a failed post rolls back
the approval).

**11. ~~Several error strings begin with a bare space~~ — FIXED** — `arcadia/rpc/core.go`, `arcadia/rpc/verdict.go`, `arcadia/rpc/transfer.go`
`" does not exist"`, `" is not pending review?"`, `" is in a team. …"`, `" is
not in a team. …"` — the entity name was dropped in an earlier refactor,
leaving a bare leading space where it used to be. Confirmed Omniplex is the
only consumer of this wire format (see the top of this doc), so restored the
missing piece rather than just trimming the space: `entityExists` now
formats `%q does not exist` with the target id, and the three RPC handlers
interpolate `m.TargetID` the same way their sibling messages already did.

**12. ~~Misspelling and inconsistent casing~~ — FIXED** —
`"[neeed to delete position]"` (§5.17) → `"[need to delete position]"`.
`"Invalid OTP Entered"` in `ResetMfaTotp` vs `"Invalid OTP entered"` in
`ActivateSession` (§5.1) → both now `"Invalid OTP entered"`. `bot_whitelist`
permission messages used `(parentheses)` where every other message uses
`[brackets]` (§5.24) → now brackets. Embed titles with intentional-by-accident
leading spaces (`" Claimed!"`, `" Unclaimed!"`, `" Approved!"`, `" Denied!"`,
`" Force Deleted!"` ×2, `" Force Certified!"`, `" Uncertified!"`,
`" Ownership Force Update!"` ×2) → leading space dropped from all of them.
Left `"__ Unverified For Futher Review!__"` alone — same class of typo but
out of scope for this pass; still frozen and tested.

**13. ~~`Authorize/Begin` does not validate or URL-encode `redirect_url` (§5.1)~~ — FIXED** — `arcadia/panel/ops_authorize.go`
It used to be interpolated raw into the Discord OAuth2 URL. `authCreateSession`
already checked the redirect against the allow-list at the token-exchange
step, so the exposure was limited to handing back a malformed or
attacker-chosen login URL — never a completed session. `authBegin` now runs
the same allow-list check up front and URL-encodes the value when building
the login URL.

**14. `UpdateChangelog` is a hard stub (§5.15)** — `arcadia/panel/ops_content.go`
Always 403, regardless of input or authentication. The DTOs still parse.

**16. ~~Vote credit tier dedup loop is broken for two or more occupants~~ — FIXED** — `arcadia/panel/ops_shop_tiers.go`
`vote_credit_tiers.position` carries a `UNIQUE ... DEFERRABLE INITIALLY
DEFERRED` constraint, which is what let the loop insert onto an occupied
position and tidy up before `COMMIT`. The loop shifted the rows found at
`index_a` to `index_b`, then set `index_a = index_b` — where `index_b` had
already been incremented *past* the rows it just wrote. It therefore never
re-checked the position it just moved a row into.

With one existing occupant this was fine. With two, it collided:

| step | table before | action | table after |
|---|---|---|---|
| create A @1 | `{}` | nothing to shift | `{A:1}` |
| create B @1 | `{A:1}` | A→2, `index_a` jumps 1→3 | `{B:1, A:2}` |
| create C @1 | `{B:1, A:2}` | B→2 (**collides with A**), `index_a` jumps 1→3 | `{C:1, B:2, A:2}` ✗ |

The deferred constraint then fired at `COMMIT` and the panel received a raw
`ERROR: duplicate key value violates unique constraint "vote_credit_tiers_position_key" (SQLSTATE 23505)`
as a 500 — which also leaked the constraint name to the client. This was
live: production had three tiers at positions 1, 2 and 3, so creating a
tier at position 1 failed outright. `EditTier` carried an identical copy of
the loop and the same defect (both now share `dedupTierPositions`).

The originally-suggested one-line patch (`index_a = index_a + 1` instead of
`index_a = index_b`) turns out not to work: re-checking the position the
loop just wrote to means the query re-finds the row it just placed there
(the query only ever excludes the target row's own id, not previously-moved
rows), which it can't distinguish from a genuine second occupant — so it
re-flags its own row as a conflict and cascades forever, even in the
"one existing occupant" case that worked before. Went with the doc's other
suggested option instead: `dedupTierPositions` is now a single set-based
`UPDATE vote_credit_tiers SET position = position + 1 WHERE position >= $1
AND id != $2`, run once. Every matching row's new position is computed from
its pre-update value in the same statement, so there's no intermediate
state for a follow-up query to misread — no loop, no cascade, no
self-conflict. `TestVoteCreditTierDedupLoop` now asserts all three creates
succeed with the tiers landing at distinct positions (3, 2, 1).

**15. `UpdateBlog` authenticated twice (§5.16)**
Upstream calls `check_auth` twice with a TODO admitting it is wasteful. Done once
here. No wire difference; the only effect is one fewer round trip and one fewer
run of the session GC.

### Fixed (internal only, no observable change)

**F1. `AddFile` leaked its temp file on hash mismatch (§5.13)** — `arcadia/panel/ops_cdn.go`
Upstream returns early on a SHA-512 mismatch without removing
`/tmp/arcadia-cdn-file…`, leaking a file up to the size of the upload while the
chunks that produced it have already been consumed. Now removed by `defer` on
every exit path. Same status codes, same bodies.

**F2. `japi_updater`'s hour-reset check underflowed (§12)** — `arcadia/tasks/bots.go`
`LAST_REFRESH - now >= 3600` on unsigned integers underflows whenever
`now > LAST_REFRESH`, which is always after the first run, so the 1800-requests
budget reset erratically. Implemented as `now - last >= 3600`.

**F3. `queue`'s previous-page handler underflowed (§11.3)** — `arcadia/bot/interactions.go`
`if current == 0 { current = 0 } current -= 1` underflows on the first page.
Both bounds are clamped.

**F4. `staff_resync` panicked on a cache miss (§12.1)** — `arcadia/tasks/staffresync.go`
The unaccounted-user branch `.unwrap()`s a `member_pos_cache` lookup that can miss
for a user present in the DB but filtered out of the cache. Treated as "no
positions" and logged.

**F5. `staff_resync` abandoned the run when its log message failed** — `arcadia/tasks/staffresync.go`

Upstream returned the error from posting to the staff log channel, which
abandoned the resync and rolled back its transaction. A staff log channel that
had been deleted — or, far more likely, a `channels.staff_logs` key missing from
config.yaml, which loads as id 0 and is only validated outside dev — therefore
stopped staff permissions syncing with Discord entirely, reported as Discord's
`10003: Unknown Channel`.

The send is now reported through the logger and stepped over. The log is an
account of what the resync did; the resync is what keeps permissions correct, and
one cannot be allowed to block the other. `impls.SendChannel` additionally names
an unconfigured channel as such rather than passing Discord's "Unknown Channel"
through for id 0.

### Not ported

`src/tasks/__toberewritten/uptime.rs` — dead code that did not compile
against the serenity API it was last touched against, and was never ported
as-is. `bots.uptime`, `bots.total_uptime` and `bots.uptime_last_checked`
are maintained again now, but not by porting this file: `popplio/bgtasks`'
`bot_uptime_check` reads presence from Popplio's own gateway cache (which
already requests the Presence intent) rather than reimplementing whatever
this task used to do.

`src/test.rs` (`modaltest`) — a dev-only command never registered in `main.rs`.

---

## (b) Security hardening

Each item below is additive: it rejects input that upstream should never have
accepted, and provably does not change behaviour for legitimate input.

**H1. CDN path containment** — `arcadia/cdnpath/cdnpath.go`, `arcadia/panel/ops_cdn.go`
`validateName`/`validatePath` are pure string checks and are the *only* thing
preventing directory escape from the CDN root. The resolved `asset_path`,
`asset_final_path` and `copy_to` are now additionally checked with
`filepath.Clean` + prefix containment against the scope root.
*Proof of no behaviour change:* any path that stays under the scope root passes
`ContainedInScope` by construction — it can only reject a path that resolves
outside the root, which the validators were already meant to forbid.
Covered by `TestContainedInScope` and `TestValidatePath`.

**H2. `PopplioStaff` proxy path validation** — `arcadia/panel/paths.go`
Upstream checks only that the path starts with `/`, so `//evil.example/x` (a
protocol-relative URL) retargets the request at another host and `..` segments
escape the API base. `safeJoinPopplio` resolves the reference against the base URL
and rejects anything carrying a scheme or host, or resolving outside the base.
*Proof of no behaviour change:* an ordinary absolute path with an optional query
string resolves to exactly the same URL string concatenation produced.

**H3. `generic_cleaner` dynamic SQL** — `arcadia/tasks/cleaners.go`
Upstream interpolates `information_schema` output straight into SQL. Table names
are now filtered to `public` schema, checked against a strict identifier pattern,
and quoted with `pgx.Identifier.Sanitize`. Entity tables and id columns come from
a fixed allow-list. The set of tables acted on is unchanged.

**H4. `asset_cleaner` dynamic SQL** — `arcadia/tasks/cleaners.go`
The table/column pairs come from a hard-coded list (never user input) and only the
id value is parameterised.

**H5. Session tokens use crypto/rand** — `arcadia/impls/crypto.go`
`impls.GenRandom` reads from `crypto/rand` over a full alphanumeric alphabet.
Popplio's existing `eureka/crypto.RandString` was deliberately *not* reused for
this: it draws from `math/rand` seeded with the wall clock and its alphabet omits
digits, which would make panel session tokens and Popplio staff tokens predictable.

**H6. `git commit -m <message>` is never shelled** — `arcadia/panel/ops_cdn.go`
`exec.Command` with separate argv elements, as upstream does. Never `sh -c`.

**H7. `asset_cleaner` dry-run** — `arcadia/tasks/cleaners.go`
The task deletes files off the CDN. `arcadia.asset_cleaner_dry_run` makes it log
what it *would* remove; every deletion is logged either way.

---

## (c) Intentional deviations

**D1. Integration into Popplio.** Per the requester's decision, the port lives
inside Popplio rather than as a standalone service. Consequences:

- **Config.** The Arcadia keys Popplio already had are read from Popplio's config
  rather than duplicated (`database_url` → `meta.postgres_url`, `token` →
  `discord_auth.token`, `frontend_url` → `sites.frontend`, `infernoplex_url` →
  `sites.infernoplex`, `popplio_url` → `sites.api`, `cdn_url` → `sites.cdn`,
  `proxy_url` → `meta.popplio_proxy`, `japi_key` → `japi.key`). What Popplio
  lacked lives under a new `arcadia:` section, and `servers`, `roles` and
  `channels` gained the staff-side ids.
  **Operational requirement:** the deployed `config.yaml` must gain these keys
  before startup, or config validation will fail. `./popplio --cmd genconfig`
  writes a fresh sample.
- **`PopplioStaff` is now a loopback call.** The panel proxies to `sites.api`,
  which is this same process. It still works — it is a real HTTP request through
  the normal auth path — but it is a round trip through the network stack to
  ourselves. If that becomes a problem the handler can be pointed at Popplio's
  chi router in-process.
- **Binary and service names** stay Popplio's (`popplio`, `popplio-staging`), not
  `bot` / `arcadia-<env>`.

**D1a. The staff bot keeps its own Discord identity.** `arcadia/dclient` owns a
second gateway connection built from `arcadia.token`, a separate Discord
application from Popplio's. Popplio's session is untouched — its intents and
caches are exactly what they were before the port.

This preserves what upstream had: mod-log embeds, role changes and audit-log
entries are attributed to the Arcadia bot, and either half can be restarted
without dropping the other's gateway. The panel OAuth app
(`arcadia.panel.client_id` / `client_secret`) was already separate from Popplio's
`discord_auth.*` and remains so.

The staff bot requests `Guilds`, `GuildMembers`, `GuildPresences` and
`GuildModeration`, and caches guilds, members, presences and roles (the panel
validates staff position role ids against the role cache). Upstream used
`GatewayIntents::all()`; only what is actually read is requested here.

A Discord failure at startup is logged and the panel still comes up — every
cache read already treats an uncached guild as "not found".

**D1b. Slash commands are the primary interface.**

- Commands are published to the main, staff and testing guilds, not just the
  staff guild. Registering per-guild rather than globally keeps staff tooling out
  of other servers and takes effect immediately instead of Discord's ~1 hour
  global propagation. The full set goes to all three; the per-command guards
  already decide where each is usable.
- Subcommands carry their options, so `/staff guildleave` and `/staff stats` are
  usable from the slash UI rather than prefix-only.
- `claim`, `unclaim`, `approve`, `deny` and `staff stats` take a **user picker**,
  matching upstream's `User`/`Member` parameter types.
- `/rpc` exposes all 18 methods as a **choice list**. Upstream autocompleted them;
  the whole set fits inside Discord's 25-choice limit, so choices need no round
  trip.
- `queue` and `claim` answer through the interaction before posting their
  component message. Posting via REST alone would have left the interaction
  unacknowledged and shown the caller "the application did not respond".

**Prefix commands are off by default**, behind `arcadia.prefix_commands`. They
still work when enabled, but reading message content needs the privileged
Message Content intent, and slash commands make it unnecessary. This is the one
config key added that has no upstream counterpart.

**D2. The panel is a second `http.Server`, not a chi route.** Popplio's global
middleware pins `Content-Type: application/json`, caps bodies at 50 MB and applies
a 30-second timeout — all three incompatible with a protocol that returns bare
text and 204s and accepts 1 GB uploads. The panel keeps its own port (3010/3011),
its own hand-rolled mux and its own middleware chain, built on `net/http` only.

**D3. Server timeouts.** Upstream set none. Here: `ReadHeaderTimeout` 30s,
`ReadTimeout`/`WriteTimeout` 30 minutes (a 1 GB upload on a slow link must not be
killed mid-flight), `IdleTimeout` 120s.

**D4. Graceful shutdown added.** The Rust version had none. The panel server
drains, the task tickers stop via context, and `arcadia.Stop` waits up to 30s.

**D5. Startup ordering.** Upstream started the panel API from inside the Discord
`Ready` handler so the cache was warm before serving. Popplio owns the connection
and starts the panel from `main`, so the panel can accept traffic before the cache
fills. Every cache read already degrades safely: dovewing falls through to
Postgres, and role/member lookups treat an uncached guild as "not found" — which
for staff-position validation means a transient
`"Role does not exist on the staff server"` during the first seconds after boot.

**D6. Dovewing.** Popplio's `eureka/dovewing` replaces Arcadia's own three-tier
cache. Same `internal_user_cache__discord` table, same 8-hour expiry, plus a Redis
hot cache. Two visible differences: Discord's `invisible` presence maps to
`offline` (Discord only ever reports invisible users as offline to third parties,
so this is not observable in practice), and the background refresh is dovewing's
bounded one rather than a detached goroutine per lookup.

**D7. OpenAPI is hand-written.** `arcadia/panel/openapi.json`, served at
`GET /openapi` as `application/json`. utoipa has no Go equivalent and a codegen
step would need maintaining in lockstep with the union codec for no gain: the API
is two routes and one union whose shape is already pinned by round-trip tests.

**D8. `info` command build metadata.** `Rustc Version` → `Go Version`; the vergen
git sha/semver/commit-message/CPU-brand/cargo-profile fields are replaced by
`debug.ReadBuildInfo`'s VCS revision and modified flag plus `GOOS/GOARCH`. Fields
with no Go equivalent report `unknown`.

**D9. Discord command framework.** poise supplied prefix+slash parsing, checks,
cooldowns, help rendering, modals and pagination. `arcadia/bot/framework.go`
reimplements the parts the commands use. Two consequences: **per-user cooldowns
are not implemented** (poise's `user_cooldown = 3` etc. have no equivalent here),
and `register` syncs guild commands directly instead of posting poise's
registration buttons.

**D10. `chrono`-compatible timestamps.** `types.Timestamp` reproduces chrono's
`SecondsFormat::AutoSi` (0, 3, 6 or 9 fractional digits). Go's own `time.Time`
marshaller trims trailing zeroes one digit at a time and would emit `.5` where
chrono emits `.500`.

**D11. Pool size.** Upstream capped its pool at 6 connections. The port shares
Popplio's existing `pgxpool`, which uses the pgx default (`max(4, numCPU)`) — a
deliberate consequence of not standing up a second pool.

**D11a. Borealis, cache servers and the HTML sanitizer are removed.** Requested
explicitly; these are deliberate feature removals, not port fidelity issues.

Gone: `arcadia.borealis_url`, `arcadia.htmlsanitize_url`, the
`borealisCacheServer` type and the `addBotToCacheServer` client.

Three things changed that a staff member or the panel can see:

1. **`core_constants.htmlsanitize_url` no longer appears in the Hello response.**
   This is a removal from a frozen wire contract — if any panel code reads it, it
   now gets `undefined`. Worth grepping the SvelteKit panel before deploying.
2. **`Approve` no longer calls Borealis** and its mod-log embed has lost the
   "Cache Server" field. The `Feedback` and `Moderator` fields are unchanged.
3. **`Approve`'s success payload changed.** It was:

   ```
   **Cache Server Invite:** https://discord.gg/<code>
   **Invite URL:** https://discord.com/api/v10/oauth2/authorize?client_id=<id>&permissions=0&scope=bot%20applications.commands&guild_id=<cache guild>
   ```

   and is now just the invite URL, without the `guild_id` that pre-selected the
   cache server:

   ```
   **Invite URL:** https://discord.com/api/v10/oauth2/authorize?client_id=<id>&permissions=0&scope=bot%20applications.commands
   ```

   The Discord `approve` reply drops "Please invite the bot to the caching server
   provided down below!" accordingly.

What did NOT change: the mod-log post still happens inside the transaction, so a
failed post still rolls the approval back. Removing Borealis also removes the
ordering hazard that note described — there is no longer an HTTP call held open
across the transaction.

**D11b. The CDN is gone entirely.** Requested explicitly, in two parts.

*Arcadia's CDN tooling (~1,070 LOC):* `UploadCdnFileChunk`, `ListCdnScopes`,
`GetMainCdnScope`, `UpdateCdnAsset` and its seven actions (ListPath, ReadFile,
CreateFolder, AddFile, CopyFile, Delete, PersistGit), the chunk cache,
`arcadia/cdnpath`, `types.Bytes` (the `Vec<u8>` number-array codec existed only
for chunk uploads), the nine `cdn.*` permissions, `arcadia.panel.cdn_scopes` /
`main_scope` / `asset_cleaner_dry_run`, and the `asset_cleaner` task. Partner
create/update no longer requires the avatar to exist, and partner delete no
longer removes an image.

*Popplio's asset pipeline:* the `assetmanager` package, `types.Asset` and
`types.AssetMetadata`, the `POST`/`DELETE /{target_type}/{target_id}/assets`
endpoints and the whole Assets route group, `sites.cdn` and `meta.cdn_path`.

Wire changes, all breaking for existing clients:

| Response | Change |
|---|---|
| `Bot`, `IndexBot` | `banner` removed |
| `Server`, `IndexServer` | `avatar`, `banner` removed |
| `Team` | `avatar`, `banner` removed |
| `Partner` | `avatar` removed |
| `Hello.core_constants` | `cdn_url` removed (with `htmlsanitize_url`) |
| `SearchEntitys` server results | `avatar` is now always `""`; upstream synthesised it from the CDN URL |
| RSS feed | channel `<image>` removed (the logo was CDN-hosted) |
| Webhook payloads | `GetAvatarURL` returns the bot's Discord avatar, or `""` for servers and teams |
| Vote embeds | `EntityInfo.Avatar` is unset for packs, teams, servers and blogs |
| SEO/OG | team and server OG images removed |
| `POST`/`DELETE /{target_type}/{target_id}/assets` | gone; nothing can upload an image any more |

Bot and user avatars still resolve: those come from dovewing, which returns
Discord's own URL. The `bp.DovewingMiddleware` that mirrored them onto the CDN
was removed, so `PlatformUser.Avatar` is now Discord's URL rather than a
self-hosted copy.

**Update:** server avatars were later reintroduced, but not by restoring the
CDN cache above. `servers.avatar` (a plain URL, `exp/add_servers_avatar.sql`)
is populated once at Add Server time from the invite resolution, then kept
fresh by Infernoplex's `serversync` task reading its own gateway cache for
every server it's a member of. `IndexServer`, `Server` and `SearchEntitys`
server results all serve it now — the `avatar` row in the table above and
line 430 no longer hold. `Team`/`Partner` avatars remain removed; nothing
analogous exists for them.

`cmd/kitehelper` still contains CDN paths in its historical migration code. That
is a separate module of one-shot migrations that are not run at startup, so it
was left alone.

**Not done: the files themselves.** Nothing deletes anything under the old
`cdn_path`, and no database column was touched. The images are still on disk and
still referenced by nothing.

**D12. `staff_resync` position ordering.** Position id sets are sorted before
being written or logged, so SQL arrays and staff-log embeds are stable between
runs. Rust's `HashSet` iteration order was arbitrary.

**D14. The permission model was replaced.** Upstream used kittycat's
`namespace.action` permissions with wildcards (`rpc.*`), negators
(`~rpc.PremiumAdd`), an `@clear` marker and position-index precedence. Popplio
now uses flat, declared permissions (`review_entities`, `manage_shop`) resolved as a
plain union of a member's roles and their extras, in the `perms` package. The
kittycat dependency is gone.

Consequences for anything speaking to this service:

- **Wire format.** `StaffMember.resolved_perms`, `staff_positions.perms` and
  `StaffEditMember.perm_overrides` are still arrays of permission strings, but
  the strings themselves are the new names. The panel's permission picker and any
  external service that reads `staff_positions` must move with it.
- **RPC methods no longer have one permission each.** `rpc.` + method name is
  replaced by `types.RPCPermission`, which maps related methods onto one
  permission — the whole claim/approve/deny loop is `review_entities`. Withholding a
  single method within a group is no longer expressible; the groups were drawn
  around what the live roles actually negated.
- **Denial is gone.** Nothing subtracts a permission any more, so a role either
  has one or does not. `exp/rewrite/flatperms.sql` reports every negator it drops, since
  a negator that was holding something back leaves its holder with that
  permission afterwards.
- **Disciplinaries.** A non-additory disciplinary now replaces the member's
  permissions outright rather than tying with their overrides at index 0 and
  letting an unstable sort decide.
- **Migration.** `exp/rewrite/flatperms.sql` converts every permission column in place and
  is verified against the catalogue by tests in `perms/migration_test.go`.

**D13. Two N+1 query patterns are batched.** The wire format is unchanged and
§5.5 explicitly permits internal batching.

- `BotQueue` and `SearchEntitys` resolved each entity's managers with two queries
  inside the loop — 2N round trips for an N-entity response. `impls.GetBotManagers`
  and `impls.GetServerManagers` resolve the same data in three queries regardless
  of N, reproducing the per-entity error messages exactly.
- `GetStaffDisciplinaries` issued one query per distinct disciplinary type; it now
  LEFT JOINs the type in the same query. A missing type still errors, as upstream's
  `fetch_one` did.

Dovewing lookups are deliberately left per-entity: they are fronted by Popplio's
Redis hot cache, so batching them would trade a cache hit for a database round
trip.

---

## Testing status

| Suite | Covers | Status |
|---|---|---|
| `arcadia/types` | Union round-trips for all 14 unions and all 18 RPC methods, `Vec<u8>` as number array, chrono timestamps, null/empty encoding, `StaffMember` serialization, wrong-shape rejection, RPC metadata completeness | **passing** |
| `arcadia/cdnpath` | Name/path validators, scope containment, granular CDN permission | **passing** |
| `arcadia/conformance` | Frozen strings across panel, rpc, tasks and bot | **passing** |
| `arcadia/panel` (unit) | Custom server: routing, CORS + preflight, response envelopes, panic recovery, body cap, listen address, chunk-cache atomicity | **passing** |
| `arcadia/panel` (integration) | Auth validators and session GC, auth failure shape, permission gates allowed/denied, dedup loop, coupon validation, ListPositions, BaseAnalytics, changelog stub, chunk upload | **passing** |
| `arcadia/dbconform` | **All 195 SQL statements PREPAREd against the real schema**, plus the runtime-composed ones | **passing** |

### Running them

```sh
# Unit + frozen-string suites
go test -ldflags=-checklinkname=0 ./arcadia/...

# Plus the database suites (point at a SCRATCH database - the fixtures write rows)
createdb arcadia_test && pg_dump -s infinity | psql -q arcadia_test
ARCADIA_TEST_DATABASE_URL=postgres://user:pass@127.0.0.1/arcadia_test \
  go test -ldflags=-checklinkname=0 ./arcadia/...
```

`-ldflags=-checklinkname=0` is required on Go 1.24+ and is **not** specific to
this port: `bytedance/sonic v1.12.2` (pulled in through `eureka/jsonimpl`) uses
`//go:linkname` against `encoding/json` internals that no longer exist, so
`go build ./...` fails for the `popplio` binary itself on a clean checkout with
`link: github.com/bytedance/sonic/ast: invalid reference to encoding/json.unquoteBytes`.
The flag disables the linkname check and both the binary and the tests build.
The real fix is upgrading `eureka`/`sonic` or building on Go ≤1.23; sonic also
prints `WARNING:(ast) sonic only supports go1.17~1.23` at startup.

### `arcadia/dbconform` — the sqlx replacement

The single most valuable suite. sqlx verified every query against the schema at
compile time; losing that was called out in §13 as the biggest regression of the
port. Instead of a hand-maintained list that would drift, `dbconform` walks the
arcadia sources with `go/ast`, extracts every SQL string literal, and `PREPARE`s
each against Postgres — which parses and plans the statement, validating every
table, column and function reference and inferring parameter types, without
executing anything.

All **195** extracted statements prepare cleanly against the production schema,
plus the eight runtime-composed statements (the `asset_cleaner` per-entity
queries and the `generic_cleaner` dynamic SQL) that are listed explicitly because
the extractor cannot see them whole.

### Still to do

- **Dovewing-dependent operations are uncovered**: `Hello`, `GetUser`,
  `BotQueue`, `SearchEntitys` and `UpdateStaffMembers` resolve a Discord user, so
  they need Redis and a live Discord session. Their SQL is covered by
  `dbconform`, but their response shapes are not exercised end to end.
- **RPC handlers are uncovered end to end**: every one of the 18 posts a mod-log
  embed, so they need a Discord session. The pipeline around them (target-type
  check, permission check, audit row, rate limit) is testable with a mocked
  Discord client and is the next thing worth adding.
- **The 12 background tasks are uncovered.** `staff_resync` in particular is
  769 lines of destructive logic and deserves a fixture-driven test.
- **No test asserts the mod-log embeds**, which are frozen but only checked as
  string literals by `arcadia/conformance`.
