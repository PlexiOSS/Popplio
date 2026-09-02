-- GENERATED FILE -- sqlc's schema source, not a migration.
--
-- sqlc parses static DDL; it does not execute the `DO $$ ... $$` guarded
-- blocks that most of db/migrations/*.sql use for idempotency (e.g.
-- `IF NOT EXISTS (SELECT ... information_schema ...) THEN ALTER TABLE ...`).
-- A table whose real shape only exists because of one of those blocks
-- (changelogs is the clearest example -- created bare by an old draft
-- migration, then evolved entirely inside DO blocks) would otherwise be
-- invisible to sqlc's simulated catalog if it replayed db/migrations/
-- directly, even though the real database has the columns.
--
-- So instead this is a `pg_dump --schema-only` snapshot of an actual
-- database that has had every migration in db/migrations/ applied and
-- passed `go run ./cmd/migrate validate` (zero drift) -- the same
-- reasoning as db/migrations/20260731235950_genesis_foundational_tables.sql,
-- applied here too: a real database's live schema is authoritative in a
-- way a static replay of idempotency-guarded migrations can't be.
--
-- Regenerate after adding/changing migrations that alter table shape:
--   1. Apply every migration to a scratch database and confirm
--      `go run ./cmd/migrate validate` reports no drift against it.
--   2. pg_dump --schema-only --no-owner --no-privileges <scratch db url> > db/schema.sql
--      (strip any `\restrict`/`\unrestrict` lines pg_dump 18+ emits --
--      they're psql meta-commands, not valid SQL statements)
--   3. Re-add this header comment block (pg_dump overwrites it).
--   4. go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
--
-- PostgreSQL database dump
--


-- Dumped from database version 18.6 (Debian 18.6-1.pgdg13+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: archive_cache_servers; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA archive_cache_servers;


--
-- Name: SCHEMA archive_cache_servers; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA archive_cache_servers IS 'Retired cache-server subsystem, archived out of public. Read-only snapshot; no application code references it. See exp/remove_cache_servers.sql.';


--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';


--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: bots_cache_server_uninvitable; Type: TABLE; Schema: archive_cache_servers; Owner: -
--

CREATE TABLE archive_cache_servers.bots_cache_server_uninvitable (
    bot_id text NOT NULL,
    cache_server_uninvitable text NOT NULL,
    archived_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE bots_cache_server_uninvitable; Type: COMMENT; Schema: archive_cache_servers; Owner: -
--

COMMENT ON TABLE archive_cache_servers.bots_cache_server_uninvitable IS 'Snapshot of bots.cache_server_uninvitable taken when the column was dropped.';


--
-- Name: alerts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alerts (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    url text,
    message text NOT NULL,
    type text NOT NULL,
    alert_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    icon text,
    title text NOT NULL,
    priority integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    acked boolean DEFAULT false NOT NULL,
    category text DEFAULT 'general'::text NOT NULL
);


--
-- Name: api_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    type text NOT NULL,
    expiry timestamp with time zone NOT NULL,
    token text NOT NULL,
    name text,
    perm_limits text[] DEFAULT '{}'::text[] NOT NULL,
    num_uses integer DEFAULT 0 NOT NULL
);


--
-- Name: apps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.apps (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    app_id text NOT NULL,
    user_id text NOT NULL,
    "position" text NOT NULL,
    review_feedback text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    questions jsonb DEFAULT '{}'::jsonb NOT NULL,
    answers jsonb DEFAULT '{}'::jsonb NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL
);


--
-- Name: automated_vote_resets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.automated_vote_resets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL
);


--
-- Name: badges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.badges (
    id text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    icon text NOT NULL,
    color text DEFAULT 'default'::text NOT NULL,
    target_types text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text NOT NULL,
    CONSTRAINT badges_color_check CHECK ((color = ANY (ARRAY['default'::text, 'success'::text, 'warning'::text, 'danger'::text, 'info'::text, 'premium'::text])))
);


--
-- Name: blacklisted_words; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blacklisted_words (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    word text NOT NULL,
    systems text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: blogs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.blogs (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    slug text NOT NULL,
    title text NOT NULL,
    description text NOT NULL,
    user_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    content text NOT NULL,
    draft boolean DEFAULT true NOT NULL,
    tags text[] NOT NULL
);


--
-- Name: bot_changelogs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bot_changelogs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    bot_id text NOT NULL,
    title text NOT NULL,
    content text DEFAULT ''::text NOT NULL,
    version text DEFAULT ''::text NOT NULL,
    created_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: bot_commands; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bot_commands (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    bot_id text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    usage text DEFAULT ''::text NOT NULL,
    category text DEFAULT ''::text NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: bot_whitelist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bot_whitelist (
    bot_id text NOT NULL,
    reason text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id text NOT NULL
);


--
-- Name: bots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bots (
    bot_id text NOT NULL,
    client_id text NOT NULL,
    tags text[] NOT NULL,
    prefix text NOT NULL,
    owner text,
    short text NOT NULL,
    long text NOT NULL,
    library text DEFAULT 'custom'::text NOT NULL,
    extra_links jsonb NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    premium boolean DEFAULT false NOT NULL,
    servers integer DEFAULT 0 NOT NULL,
    shards integer DEFAULT 0 NOT NULL,
    users integer DEFAULT 0 NOT NULL,
    clicks integer DEFAULT 0 NOT NULL,
    invite_clicks integer DEFAULT 0 NOT NULL,
    invite text NOT NULL,
    type text DEFAULT 'pending'::text NOT NULL,
    vote_banned boolean DEFAULT false NOT NULL,
    start_premium_period timestamp with time zone DEFAULT now() NOT NULL,
    premium_period_length interval DEFAULT '12:00:00'::interval NOT NULL,
    cert_reason text,
    uptime bigint DEFAULT 0 NOT NULL,
    total_uptime bigint DEFAULT 0 NOT NULL,
    claimed_by text,
    approval_note text DEFAULT 'No note'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    unique_clicks text[] DEFAULT '{}'::text[] NOT NULL,
    api_token text DEFAULT public.uuid_generate_v4() NOT NULL,
    last_claimed timestamp with time zone,
    team_owner uuid,
    shard_list bigint[] DEFAULT '{}'::bigint[] NOT NULL,
    captcha_opt_out boolean DEFAULT false NOT NULL,
    uptime_last_checked timestamp with time zone DEFAULT now() NOT NULL,
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    last_stats_post timestamp with time zone,
    vanity_ref uuid NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    approximate_votes integer DEFAULT 0 NOT NULL,
    last_japi_update timestamp with time zone,
    flags text[] DEFAULT '{}'::text[],
    self_status text,
    boosted_until timestamp with time zone,
    featured_until timestamp with time zone,
    supporter_badge boolean DEFAULT false NOT NULL,
    vote_blitz_until timestamp with time zone,
    moderation_categories text[] DEFAULT '{}'::text[] NOT NULL,
    moderation_flagged boolean DEFAULT false NOT NULL,
    moderation_checked_at timestamp with time zone,
    spotlighted_until timestamp with time zone,
    CONSTRAINT bots_self_status_check CHECK (((self_status IS NULL) OR (self_status = ANY (ARRAY['online'::text, 'idle'::text, 'dnd'::text, 'offline'::text])))),
    CONSTRAINT owner_team_oneof CHECK (((owner IS NULL) <> (team_owner IS NULL)))
);


--
-- Name: changelogs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelogs (
    version text NOT NULL,
    added text[] NOT NULL,
    updated text[] NOT NULL,
    removed text[] NOT NULL,
    extra_description text DEFAULT ''::text NOT NULL,
    prerelease boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    itag uuid DEFAULT gen_random_uuid() NOT NULL,
    project text DEFAULT 'popplio'::text NOT NULL,
    published boolean DEFAULT false NOT NULL,
    created_by text DEFAULT 'unknown'::text NOT NULL,
    fixed text[] DEFAULT '{}'::text[] NOT NULL,
    CONSTRAINT changelogs_project_check CHECK ((project = ANY (ARRAY['popplio'::text, 'omniplex'::text, 'keel'::text])))
);


--
-- Name: entity_badges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_badges (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    badge_id text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    awarded_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: entity_vote_redeem_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_vote_redeem_logs (
    target_id text NOT NULL,
    target_type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    redeemed_at timestamp with time zone,
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    redeemed_credits integer DEFAULT 0 NOT NULL,
    credits integer NOT NULL
);


--
-- Name: entity_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_votes (
    itag uuid DEFAULT public.uuid_generate_v4() CONSTRAINT entity_votes_id_not_null NOT NULL,
    target_id text NOT NULL,
    target_type text NOT NULL,
    author text NOT NULL,
    upvote boolean NOT NULL,
    void boolean DEFAULT false NOT NULL,
    void_reason text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    vote_num integer DEFAULT 0 NOT NULL,
    voided_at timestamp with time zone,
    immutable boolean DEFAULT false NOT NULL,
    credit_redeem uuid
);


--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.goose_db_version ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: internal_user_cache__discord; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.internal_user_cache__discord (
    id text NOT NULL,
    username text NOT NULL,
    display_name text NOT NULL,
    avatar text NOT NULL,
    bot boolean NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: mod_cases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mod_cases (
    case_id bigint NOT NULL,
    guild_id text NOT NULL,
    user_id text NOT NULL,
    moderator_id text NOT NULL,
    action text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mod_cases_action_check CHECK ((action = ANY (ARRAY['kick'::text, 'ban'::text, 'timeout'::text, 'warn'::text, 'purge'::text, 'lock'::text, 'unlock'::text, 'automod'::text])))
);


--
-- Name: TABLE mod_cases; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.mod_cases IS 'Durable moderation history backing /modlogs. Written by arcadia/bot on every moderation action, including automod. See exp/rewrite/mod_cases.sql.';


--
-- Name: mod_cases_case_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.mod_cases ALTER COLUMN case_id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.mod_cases_case_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: pack_emojis; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pack_emojis (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pack_url text NOT NULL,
    name text NOT NULL,
    animated boolean DEFAULT false NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    downloads integer DEFAULT 0 NOT NULL
);


--
-- Name: pack_stickers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pack_stickers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pack_url text NOT NULL,
    name text NOT NULL,
    animated boolean DEFAULT false NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    downloads integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: packs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.packs (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    owner text NOT NULL,
    name text DEFAULT 'My pack'::text NOT NULL,
    short text NOT NULL,
    tags text[] NOT NULL,
    url text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    bots text[] NOT NULL,
    votes integer DEFAULT 0 NOT NULL,
    vote_banned boolean DEFAULT false NOT NULL,
    servers text[] DEFAULT '{}'::text[] NOT NULL,
    pack_type text DEFAULT 'bot'::text NOT NULL,
    CONSTRAINT packs_pack_type_check CHECK ((pack_type = ANY (ARRAY['bot'::text, 'server'::text, 'emoji'::text, 'sticker'::text])))
);


--
-- Name: partner_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.partner_types (
    id text NOT NULL,
    name text NOT NULL,
    short text NOT NULL,
    icon text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: partners; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.partners (
    id text NOT NULL,
    name text NOT NULL,
    short text NOT NULL,
    user_id text NOT NULL,
    image text NOT NULL,
    links jsonb NOT NULL,
    type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    bot_id text
);


--
-- Name: reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reports (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    reporter_id text NOT NULL,
    reason text NOT NULL,
    description text NOT NULL,
    status text DEFAULT 'open'::text NOT NULL,
    resolved_by text,
    resolution_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    resolved_at timestamp with time zone,
    CONSTRAINT reports_status_check CHECK ((status = ANY (ARRAY['open'::text, 'under_review'::text, 'resolved'::text, 'dismissed'::text])))
);


--
-- Name: reviews; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.reviews (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    author text NOT NULL,
    content text DEFAULT 'Very good bot!'::text NOT NULL,
    stars integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    parent_id uuid,
    target_type text NOT NULL,
    target_id text NOT NULL,
    owner_review boolean DEFAULT false NOT NULL
);


--
-- Name: rpc_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rpc_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    method text NOT NULL,
    data jsonb NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: server_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.server_templates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    short text NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    owner text NOT NULL,
    usage_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    channels jsonb DEFAULT '[]'::jsonb NOT NULL,
    roles jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: server_template_reactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.server_template_reactions (
    template_id uuid NOT NULL,
    user_id text NOT NULL,
    liked boolean NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: servers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.servers (
    server_id text NOT NULL,
    name text NOT NULL,
    total_members integer NOT NULL,
    online_members integer NOT NULL,
    invite text NOT NULL,
    team_owner uuid NOT NULL,
    short text NOT NULL,
    long text NOT NULL,
    extra_links jsonb NOT NULL,
    state text DEFAULT 'public'::text NOT NULL,
    clicks integer DEFAULT 0 NOT NULL,
    invite_clicks integer DEFAULT 0 NOT NULL,
    unique_clicks text[] DEFAULT '{}'::text[] NOT NULL,
    vanity_ref uuid NOT NULL,
    type text DEFAULT 'pending'::text NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    premium boolean DEFAULT false NOT NULL,
    start_premium_period timestamp with time zone DEFAULT now() NOT NULL,
    premium_period_length interval DEFAULT '12:00:00'::interval NOT NULL,
    vote_banned boolean DEFAULT false NOT NULL,
    captcha_opt_out boolean DEFAULT false NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_claimed timestamp with time zone,
    claimed_by text,
    approximate_votes integer DEFAULT 0 NOT NULL,
    blacklisted_users text[] DEFAULT '{}'::text[] NOT NULL,
    login_required_for_invite boolean DEFAULT false NOT NULL,
    show_emojis boolean DEFAULT false NOT NULL,
    emojis jsonb DEFAULT '[]'::jsonb NOT NULL,
    stickers jsonb DEFAULT '[]'::jsonb NOT NULL,
    emojis_synced_at timestamp with time zone,
    avatar text DEFAULT ''::text NOT NULL,
    boosted_until timestamp with time zone,
    featured_until timestamp with time zone,
    supporter_badge boolean DEFAULT false NOT NULL,
    vote_blitz_until timestamp with time zone,
    approval_note text DEFAULT ''::text NOT NULL,
    stats_self_managed boolean DEFAULT false NOT NULL,
    last_stats_post timestamp with time zone,
    discord_nsfw_level smallint DEFAULT 0 NOT NULL,
    nsfw_channel_count integer DEFAULT 0 NOT NULL,
    moderation_categories text[] DEFAULT '{}'::text[] NOT NULL,
    moderation_flagged boolean DEFAULT false NOT NULL,
    moderation_checked_at timestamp with time zone,
    spotlighted_until timestamp with time zone
);


--
-- Name: shop_coupon_redemptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_coupon_redemptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    coupon_id text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    redeemed_by text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: shop_coupons; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_coupons (
    id text NOT NULL,
    code text NOT NULL,
    public boolean DEFAULT false NOT NULL,
    max_uses integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text NOT NULL,
    reuse_wait_duration integer,
    expiry integer,
    applicable_items text[] NOT NULL,
    requirements text[] DEFAULT '{}'::text[] NOT NULL,
    allowed_users text[] DEFAULT '{}'::text[] NOT NULL,
    usable boolean DEFAULT false NOT NULL,
    target_types text[] DEFAULT '{}'::text[] NOT NULL,
    cents double precision DEFAULT 0
);


--
-- Name: shop_holds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_holds (
    target_id text NOT NULL,
    target_type text NOT NULL,
    item text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    duration interval,
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);


--
-- Name: shop_item_benefits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_item_benefits (
    id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    created_by text NOT NULL,
    updated_by text NOT NULL,
    target_types text[] NOT NULL
);


--
-- Name: shop_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_items (
    id text NOT NULL,
    name text NOT NULL,
    cents double precision NOT NULL,
    target_types text[] NOT NULL,
    benefits text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    created_by text NOT NULL,
    updated_by text NOT NULL,
    duration integer NOT NULL,
    description text NOT NULL
);


--
-- Name: shop_purchases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.shop_purchases (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    item_id text NOT NULL,
    cents double precision NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: staff_disciplinary; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_disciplinary (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expiry interval,
    title text NOT NULL,
    description text NOT NULL,
    type text NOT NULL,
    state text NOT NULL
);


--
-- Name: staff_disciplinary_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_disciplinary_types (
    id text NOT NULL,
    name text NOT NULL,
    self_assignable boolean DEFAULT false NOT NULL,
    perm_limits text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    additory boolean DEFAULT false NOT NULL,
    needs_approval boolean NOT NULL,
    max_expiry interval,
    description text NOT NULL
);


--
-- Name: staff_general_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_general_logs (
    user_id text NOT NULL,
    action text NOT NULL,
    data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: staff_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_members (
    user_id text NOT NULL,
    perm_overrides text[] DEFAULT '{}'::text[] NOT NULL,
    mfa_secret text,
    no_autosync boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    positions uuid[] NOT NULL,
    mfa_verified boolean DEFAULT false NOT NULL,
    unaccounted boolean DEFAULT false NOT NULL
);


--
-- Name: staff_onboardings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_onboardings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    guild_id text NOT NULL,
    void boolean DEFAULT false NOT NULL,
    questions jsonb,
    answers jsonb,
    verdict jsonb,
    staff_verify_code text
);


--
-- Name: staff_positions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_positions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    role_id text NOT NULL,
    perms text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    index integer NOT NULL,
    corresponding_roles jsonb DEFAULT '[]'::jsonb NOT NULL,
    icon text DEFAULT 'mdi:user'::text NOT NULL,
    CONSTRAINT name_rid_not_empty CHECK (((name <> ''::text) AND (role_id <> ''::text))),
    CONSTRAINT staff_positions_index_check CHECK ((index > 0))
);


--
-- Name: staff_template_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_template_types (
    id text NOT NULL,
    name text NOT NULL,
    icon text NOT NULL,
    short text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: staff_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staff_templates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    emoji text NOT NULL,
    tags text[] NOT NULL,
    description text NOT NULL,
    type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    entity_type text DEFAULT 'bot'::text NOT NULL
);


--
-- Name: staffpanel__authchain; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.staffpanel__authchain (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    token text NOT NULL,
    popplio_token text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    task_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    task_key text,
    task_name text NOT NULL,
    output jsonb,
    statuses jsonb[] DEFAULT '{}'::jsonb[] NOT NULL,
    for_user text,
    allow_unauthenticated boolean DEFAULT false NOT NULL,
    expiry interval NOT NULL,
    state text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: team_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_members (
    user_id text NOT NULL,
    team_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    flags text[] DEFAULT '{}'::text[] NOT NULL,
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    mentionable boolean DEFAULT true NOT NULL,
    data_holder boolean DEFAULT false NOT NULL,
    service text DEFAULT 'api'::text NOT NULL
);


--
-- Name: teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teams (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name text NOT NULL,
    short text,
    tags text[],
    approximate_votes integer DEFAULT 0 NOT NULL,
    extra_links jsonb DEFAULT '[]'::jsonb NOT NULL,
    vote_banned boolean DEFAULT false NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    vanity_ref uuid NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    service text DEFAULT 'api'::text NOT NULL
);


--
-- Name: tickets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tickets (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    channel_id text NOT NULL,
    topic_id text NOT NULL,
    topic jsonb DEFAULT '{}'::jsonb NOT NULL,
    issue text NOT NULL,
    ticket_context jsonb DEFAULT '{}'::jsonb NOT NULL,
    messages jsonb DEFAULT '{}'::jsonb NOT NULL,
    user_id text NOT NULL,
    id text NOT NULL,
    close_user_id text,
    open boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    enc_key text
);


--
-- Name: user_notification_prefs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_notification_prefs (
    user_id text NOT NULL,
    category text NOT NULL,
    enabled boolean DEFAULT true NOT NULL
);


--
-- Name: user_notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_notifications (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    notif_id text NOT NULL,
    auth text NOT NULL,
    p256dh text NOT NULL,
    endpoint text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    ua text DEFAULT ''::text NOT NULL
);


--
-- Name: user_reminders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_reminders (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    target_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_acked timestamp with time zone DEFAULT now() NOT NULL,
    target_type text NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id text NOT NULL,
    experiments text[] DEFAULT '{}'::text[] NOT NULL,
    certified boolean DEFAULT false NOT NULL,
    developer boolean DEFAULT false NOT NULL,
    captcha_sponsor_enabled boolean DEFAULT true NOT NULL,
    extra_links jsonb DEFAULT '[]'::jsonb NOT NULL,
    api_token text NOT NULL,
    about text DEFAULT 'I am a very mysterious person'::text,
    vote_banned boolean DEFAULT false NOT NULL,
    banned boolean DEFAULT false NOT NULL,
    bug_hunters boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_booster_claim timestamp with time zone,
    app_banned boolean DEFAULT false NOT NULL
);


--
-- Name: vanity; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vanity (
    itag uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_id text NOT NULL,
    target_type text NOT NULL,
    code public.citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: vote_credit_tiers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vote_credit_tiers (
    id text NOT NULL,
    "position" integer NOT NULL,
    votes integer NOT NULL,
    cents double precision NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    target_type text NOT NULL,
    CONSTRAINT vote_credit_tiers_position_check CHECK (("position" > 0)),
    CONSTRAINT vote_credit_tiers_votes_check CHECK ((votes >= 0))
);


--
-- Name: webhook_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhook_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_id text NOT NULL,
    target_type text NOT NULL,
    user_id text NOT NULL,
    url text NOT NULL,
    data jsonb NOT NULL,
    bad_intent boolean NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    state text DEFAULT 'PENDING'::text NOT NULL,
    tries integer DEFAULT 0 NOT NULL,
    last_try timestamp with time zone DEFAULT now() NOT NULL,
    response text,
    status_code integer DEFAULT 0 NOT NULL,
    webhook_id uuid NOT NULL,
    request_headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    response_headers jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    target_id text NOT NULL,
    target_type text NOT NULL,
    url text NOT NULL,
    secret text NOT NULL,
    broken boolean DEFAULT false NOT NULL,
    simple_auth boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    event_whitelist text[] DEFAULT '{}'::text[] NOT NULL,
    name text DEFAULT 'My untitled webhook'::text NOT NULL,
    failed_requests integer DEFAULT 0 NOT NULL,
    hmac_auth boolean DEFAULT false NOT NULL,
    CONSTRAINT webhooks_secret_check CHECK ((secret <> ''::text)),
    CONSTRAINT webhooks_url_check CHECK ((url <> ''::text))
);


--
-- Name: bots_cache_server_uninvitable bots_cache_server_uninvitable_pkey; Type: CONSTRAINT; Schema: archive_cache_servers; Owner: -
--

ALTER TABLE ONLY archive_cache_servers.bots_cache_server_uninvitable
    ADD CONSTRAINT bots_cache_server_uninvitable_pkey PRIMARY KEY (bot_id);


--
-- Name: alerts alerts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT alerts_pkey PRIMARY KEY (itag);


--
-- Name: api_sessions api_token_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_sessions
    ADD CONSTRAINT api_token_unique UNIQUE (token);


--
-- Name: apps apps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (itag);


--
-- Name: automated_vote_resets automated_vote_resets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.automated_vote_resets
    ADD CONSTRAINT automated_vote_resets_pkey PRIMARY KEY (id);


--
-- Name: badges badges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.badges
    ADD CONSTRAINT badges_pkey PRIMARY KEY (id);


--
-- Name: blacklisted_words blacklisted_words_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blacklisted_words
    ADD CONSTRAINT blacklisted_words_pkey PRIMARY KEY (id);


--
-- Name: blogs blogs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blogs
    ADD CONSTRAINT blogs_pkey PRIMARY KEY (itag);


--
-- Name: blogs blogs_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blogs
    ADD CONSTRAINT blogs_slug_key UNIQUE (slug);


--
-- Name: bot_changelogs bot_changelogs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_changelogs
    ADD CONSTRAINT bot_changelogs_pkey PRIMARY KEY (id);


--
-- Name: bot_commands bot_commands_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_commands
    ADD CONSTRAINT bot_commands_pkey PRIMARY KEY (id);


--
-- Name: bot_whitelist bot_whitelist_bot_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_whitelist
    ADD CONSTRAINT bot_whitelist_bot_id_key UNIQUE (bot_id);


--
-- Name: bots botid_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT botid_unique UNIQUE (bot_id);


--
-- Name: bots bots_bot_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_bot_id_key UNIQUE (bot_id);


--
-- Name: bots bots_client_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_client_id_key UNIQUE (client_id);


--
-- Name: bots bots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_pkey PRIMARY KEY (itag);


--
-- Name: changelogs changelogs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelogs
    ADD CONSTRAINT changelogs_pkey PRIMARY KEY (itag);


--
-- Name: changelogs changelogs_project_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelogs
    ADD CONSTRAINT changelogs_project_version_key UNIQUE (project, version);


--
-- Name: entity_badges entity_badges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_badges
    ADD CONSTRAINT entity_badges_pkey PRIMARY KEY (itag);


--
-- Name: entity_badges entity_badges_target_type_target_id_badge_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_badges
    ADD CONSTRAINT entity_badges_target_type_target_id_badge_id_key UNIQUE (target_type, target_id, badge_id);


--
-- Name: entity_vote_redeem_logs entity_vote_redeem_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_vote_redeem_logs
    ADD CONSTRAINT entity_vote_redeem_logs_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);


--
-- Name: webhooks id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT id_unique UNIQUE (id);


--
-- Name: internal_user_cache__discord internal_user_cache__discord_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.internal_user_cache__discord
    ADD CONSTRAINT internal_user_cache__discord_pkey PRIMARY KEY (id);


--
-- Name: mod_cases mod_cases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mod_cases
    ADD CONSTRAINT mod_cases_pkey PRIMARY KEY (case_id);


--
-- Name: staff_positions name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_positions
    ADD CONSTRAINT name_unique UNIQUE (name);


--
-- Name: pack_emojis pack_emojis_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pack_emojis
    ADD CONSTRAINT pack_emojis_pkey PRIMARY KEY (id);


--
-- Name: pack_stickers pack_stickers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pack_stickers
    ADD CONSTRAINT pack_stickers_pkey PRIMARY KEY (id);


--
-- Name: packs packages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.packs
    ADD CONSTRAINT packages_pkey PRIMARY KEY (itag);


--
-- Name: packs packages_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.packs
    ADD CONSTRAINT packages_url_key UNIQUE (url);


--
-- Name: partner_types partner_types_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partner_types
    ADD CONSTRAINT partner_types_name_key UNIQUE (name);


--
-- Name: partner_types partner_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partner_types
    ADD CONSTRAINT partner_types_pkey PRIMARY KEY (id);


--
-- Name: partners partners_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partners
    ADD CONSTRAINT partners_name_key UNIQUE (name);


--
-- Name: partners partners_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partners
    ADD CONSTRAINT partners_pkey PRIMARY KEY (id);


--
-- Name: user_notifications poppypaw_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notifications
    ADD CONSTRAINT poppypaw_pkey PRIMARY KEY (itag);


--
-- Name: reports reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (id);


--
-- Name: reviews reviews_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_id_key UNIQUE (id);


--
-- Name: reviews reviews_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT reviews_pkey PRIMARY KEY (itag);


--
-- Name: staff_positions role_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_positions
    ADD CONSTRAINT role_id_unique UNIQUE (role_id);


--
-- Name: rpc_logs rpc_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rpc_logs
    ADD CONSTRAINT rpc_logs_pkey PRIMARY KEY (id);


--
-- Name: server_templates server_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_templates
    ADD CONSTRAINT server_templates_pkey PRIMARY KEY (id);


--
-- Name: server_template_reactions server_template_reactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_template_reactions
    ADD CONSTRAINT server_template_reactions_pkey PRIMARY KEY (template_id, user_id);


--
-- Name: servers servers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servers
    ADD CONSTRAINT servers_pkey PRIMARY KEY (server_id);


--
-- Name: shop_holds shop_holds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_holds
    ADD CONSTRAINT shop_holds_pkey PRIMARY KEY (id);


--
-- Name: shop_coupon_redemptions shop_coupon_redemptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_coupon_redemptions
    ADD CONSTRAINT shop_coupon_redemptions_pkey PRIMARY KEY (id);


--
-- Name: shop_coupons shop_coupons_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_coupons
    ADD CONSTRAINT shop_coupons_pkey PRIMARY KEY (id);


--
-- Name: shop_item_benefits shop_item_benefits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_item_benefits
    ADD CONSTRAINT shop_item_benefits_pkey PRIMARY KEY (id);


--
-- Name: shop_items shop_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_items
    ADD CONSTRAINT shop_items_pkey PRIMARY KEY (id);


--
-- Name: shop_purchases shop_purchases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_purchases
    ADD CONSTRAINT shop_purchases_pkey PRIMARY KEY (id);


--
-- Name: user_reminders silverpelt_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_reminders
    ADD CONSTRAINT silverpelt_pkey PRIMARY KEY (itag);


--
-- Name: staff_disciplinary_types staff_disciplinary_actions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_disciplinary_types
    ADD CONSTRAINT staff_disciplinary_actions_pkey PRIMARY KEY (id);


--
-- Name: staff_disciplinary staff_disciplinary_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_disciplinary
    ADD CONSTRAINT staff_disciplinary_pkey PRIMARY KEY (id);


--
-- Name: staff_onboardings staff_onboardings_guild_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_onboardings
    ADD CONSTRAINT staff_onboardings_guild_id_key UNIQUE (guild_id);


--
-- Name: staff_onboardings staff_onboardings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_onboardings
    ADD CONSTRAINT staff_onboardings_pkey PRIMARY KEY (id);


--
-- Name: staff_positions staff_positions_index_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_positions
    ADD CONSTRAINT staff_positions_index_unique UNIQUE (index) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: staff_positions staff_positions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_positions
    ADD CONSTRAINT staff_positions_pkey PRIMARY KEY (id);


--
-- Name: staff_template_types staff_template_types_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_template_types
    ADD CONSTRAINT staff_template_types_name_key UNIQUE (name);


--
-- Name: staff_template_types staff_template_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_template_types
    ADD CONSTRAINT staff_template_types_pkey PRIMARY KEY (id);


--
-- Name: staff_templates staff_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_templates
    ADD CONSTRAINT staff_templates_pkey PRIMARY KEY (id);


--
-- Name: staffpanel__authchain staffpanel__authchain_itag_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staffpanel__authchain
    ADD CONSTRAINT staffpanel__authchain_itag_key UNIQUE (itag);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (task_id);


--
-- Name: team_members team_member_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_member_unique UNIQUE (team_id, user_id);


--
-- Name: team_members team_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_pkey PRIMARY KEY (itag);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (id);


--
-- Name: tickets tickets2_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets2_id_key UNIQUE (id);


--
-- Name: tickets tickets2_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets2_pkey PRIMARY KEY (itag);


--
-- Name: user_reminders user_id_target_type_target_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_reminders
    ADD CONSTRAINT user_id_target_type_target_id_unique UNIQUE (target_id, target_type, user_id);


--
-- Name: staff_members user_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_members
    ADD CONSTRAINT user_id_unique UNIQUE (user_id);


--
-- Name: user_notification_prefs user_notification_prefs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_prefs
    ADD CONSTRAINT user_notification_prefs_pkey PRIMARY KEY (user_id, category);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (itag);


--
-- Name: users users_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_user_id_key UNIQUE (user_id);


--
-- Name: vanity vanity_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vanity
    ADD CONSTRAINT vanity_code_key UNIQUE (code);


--
-- Name: vanity vanity_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vanity
    ADD CONSTRAINT vanity_pkey PRIMARY KEY (itag);


--
-- Name: vanity vanity_target_id_target_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vanity
    ADD CONSTRAINT vanity_target_id_target_type_key UNIQUE (target_id, target_type);


--
-- Name: vote_credit_tiers vote_credit_tiers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vote_credit_tiers
    ADD CONSTRAINT vote_credit_tiers_pkey PRIMARY KEY (id);


--
-- Name: vote_credit_tiers vote_credit_tiers_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vote_credit_tiers
    ADD CONSTRAINT vote_credit_tiers_position_key UNIQUE ("position") DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhook_logs webhook_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_target_id_target_type_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_target_id_target_type_key UNIQUE (target_id, target_type);


--
-- Name: blacklisted_words word_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blacklisted_words
    ADD CONSTRAINT word_unique UNIQUE (word);


--
-- Name: bot_changelogs_bot_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bot_changelogs_bot_idx ON public.bot_changelogs USING btree (bot_id, created_at DESC);


--
-- Name: bot_commands_bot_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bot_commands_bot_idx ON public.bot_commands USING btree (bot_id, "position");


--
-- Name: bots_short_fts_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bots_short_fts_idx ON public.bots USING gin (to_tsvector('english'::regconfig, short));


--
-- Name: changelogs_project_published_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changelogs_project_published_idx ON public.changelogs USING btree (project, published, created_at DESC);


--
-- Name: entity_badges_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX entity_badges_target_idx ON public.entity_badges USING btree (target_type, target_id);


--
-- Name: entity_votes_target_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX entity_votes_target_created_idx ON public.entity_votes USING btree (target_type, target_id, created_at) WHERE (void = false);


--
-- Name: mod_cases_guild_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX mod_cases_guild_user_idx ON public.mod_cases USING btree (guild_id, user_id, created_at DESC);


--
-- Name: pack_emojis_pack_url_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pack_emojis_pack_url_idx ON public.pack_emojis USING btree (pack_url);


--
-- Name: pack_stickers_pack_url_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pack_stickers_pack_url_idx ON public.pack_stickers USING btree (pack_url);


--
-- Name: positions_card; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX positions_card ON public.staff_members USING btree (cardinality(positions));


--
-- Name: reports_reporter_target_open_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX reports_reporter_target_open_idx ON public.reports USING btree (target_type, target_id, reporter_id) WHERE (status = 'open'::text);


--
-- Name: reports_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reports_status_idx ON public.reports USING btree (status);


--
-- Name: reports_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX reports_target_idx ON public.reports USING btree (target_type, target_id);


--
-- Name: server_templates_code_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX server_templates_code_key ON public.server_templates USING btree (code);


--
-- Name: server_templates_owner_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX server_templates_owner_idx ON public.server_templates USING btree (owner);


--
-- Name: servers_name_fts_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX servers_name_fts_idx ON public.servers USING gin (to_tsvector('english'::regconfig, name));


--
-- Name: servers_name_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX servers_name_trgm_idx ON public.servers USING gin (name public.gin_trgm_ops);


--
-- Name: servers_short_fts_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX servers_short_fts_idx ON public.servers USING gin (to_tsvector('english'::regconfig, short));


--
-- Name: shop_coupon_redemptions_coupon_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX shop_coupon_redemptions_coupon_id_idx ON public.shop_coupon_redemptions USING btree (coupon_id);


--
-- Name: shop_coupon_redemptions_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX shop_coupon_redemptions_target_idx ON public.shop_coupon_redemptions USING btree (target_type, target_id);


--
-- Name: shop_purchases_target_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX shop_purchases_target_idx ON public.shop_purchases USING btree (target_type, target_id);


--
-- Name: ur_uid_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ur_uid_key ON public.user_reminders USING btree (user_id, target_id, target_type);


--
-- Name: web_api_tokens_expiry_token_id_target_type_target_id_type_p_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX web_api_tokens_expiry_token_id_target_type_target_id_type_p_idx ON public.api_sessions USING btree (expiry, token, id, target_type, target_id, type, perm_limits, name, created_at);


--
-- Name: webhook_logs_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_idx ON public.webhook_logs USING btree (user_id);


--
-- Name: reviews author_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.reviews
    ADD CONSTRAINT author_fkey FOREIGN KEY (author) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: bot_changelogs bot_changelogs_bot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_changelogs
    ADD CONSTRAINT bot_changelogs_bot_id_fkey FOREIGN KEY (bot_id) REFERENCES public.bots(bot_id) ON DELETE CASCADE;


--
-- Name: bot_commands bot_commands_bot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bot_commands
    ADD CONSTRAINT bot_commands_bot_id_fkey FOREIGN KEY (bot_id) REFERENCES public.bots(bot_id) ON DELETE CASCADE;


--
-- Name: bots bots_team_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_team_owner_fkey FOREIGN KEY (team_owner) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: bots bots_vanity_ref_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: entity_badges entity_badges_badge_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_badges
    ADD CONSTRAINT entity_badges_badge_id_fkey FOREIGN KEY (badge_id) REFERENCES public.badges(id) ON DELETE CASCADE;


--
-- Name: entity_votes entity_votes_author_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_votes
    ADD CONSTRAINT entity_votes_author_fkey FOREIGN KEY (author) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: entity_votes entity_votes_credit_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_votes
    ADD CONSTRAINT entity_votes_credit_fkey FOREIGN KEY (credit_redeem) REFERENCES public.entity_vote_redeem_logs(id) ON UPDATE SET NULL ON DELETE SET NULL;


--
-- Name: bots owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT owner_fkey FOREIGN KEY (owner) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: packs owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.packs
    ADD CONSTRAINT owner_fkey FOREIGN KEY (owner) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: pack_emojis pack_emojis_pack_url_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pack_emojis
    ADD CONSTRAINT pack_emojis_pack_url_fkey FOREIGN KEY (pack_url) REFERENCES public.packs(url) ON DELETE CASCADE;


--
-- Name: pack_stickers pack_stickers_pack_url_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pack_stickers
    ADD CONSTRAINT pack_stickers_pack_url_fkey FOREIGN KEY (pack_url) REFERENCES public.packs(url) ON DELETE CASCADE;


--
-- Name: server_template_reactions server_template_reactions_template_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.server_template_reactions
    ADD CONSTRAINT server_template_reactions_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.server_templates(id) ON DELETE CASCADE;


--
-- Name: partners partners_bot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partners
    ADD CONSTRAINT partners_bot_id_fkey FOREIGN KEY (bot_id) REFERENCES public.bots(bot_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: partners partners_type_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partners
    ADD CONSTRAINT partners_type_fkey FOREIGN KEY (type) REFERENCES public.partner_types(id);


--
-- Name: partners partners_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.partners
    ADD CONSTRAINT partners_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: rpc_logs rpc_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rpc_logs
    ADD CONSTRAINT rpc_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: servers servers_team_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servers
    ADD CONSTRAINT servers_team_owner_fkey FOREIGN KEY (team_owner) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: servers servers_vanity_ref_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.servers
    ADD CONSTRAINT servers_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: shop_coupon_redemptions shop_coupon_redemptions_coupon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_coupon_redemptions
    ADD CONSTRAINT shop_coupon_redemptions_coupon_id_fkey FOREIGN KEY (coupon_id) REFERENCES public.shop_coupons(id) ON DELETE CASCADE;


--
-- Name: shop_holds shop_holds_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_holds
    ADD CONSTRAINT shop_holds_item_fkey FOREIGN KEY (item) REFERENCES public.shop_items(id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: shop_purchases shop_purchases_item_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_purchases
    ADD CONSTRAINT shop_purchases_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.shop_items(id);


--
-- Name: staff_disciplinary staff_disciplinary_type_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_disciplinary
    ADD CONSTRAINT staff_disciplinary_type_fkey FOREIGN KEY (type) REFERENCES public.staff_disciplinary_types(id);


--
-- Name: staff_disciplinary staff_disciplinary_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_disciplinary
    ADD CONSTRAINT staff_disciplinary_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: staff_general_logs staff_general_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_general_logs
    ADD CONSTRAINT staff_general_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: staff_members staff_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_members
    ADD CONSTRAINT staff_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: staff_onboardings staff_onboardings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_onboardings
    ADD CONSTRAINT staff_onboardings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: staff_templates staff_templates_type_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_templates
    ADD CONSTRAINT staff_templates_type_fkey FOREIGN KEY (type) REFERENCES public.staff_template_types(id);


--
-- Name: staffpanel__authchain staffpanel__authchain_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staffpanel__authchain
    ADD CONSTRAINT staffpanel__authchain_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: team_members team_members_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: team_members team_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: teams teams_vanity_ref_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag);


--
-- Name: tickets tickets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tickets
    ADD CONSTRAINT tickets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE NOT VALID;


--
-- Name: alerts user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alerts
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: apps user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.apps
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: blogs user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blogs
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_notifications user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notifications
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_reminders user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_reminders
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhook_logs webhook_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhook_logs webhook_logs_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--


