-- +goose Up
-- +goose StatementBegin
-- The 33 tables below predate ALL migration tracking in this repo --
-- no exp/*.sql file, no kitehelper migration, nothing creates them.
-- They were set up some other way before Popplio's schema history was
-- ever captured anywhere. Discovered by actually running every ported
-- migration against a fresh database and hitting missing-table errors
-- (entity_votes references users, which nothing created).
--
-- Captured via pg_dump --schema-only against prod (2026-08-28), not
-- hand-transcribed -- exact column types, defaults, constraints,
-- indexes, and foreign keys, not a best-effort reconstruction.
-- __dp_mfa was deliberately excluded: confirmed deprecated/unused,
-- not part of Popplio's own schema.
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
-- pg_dump normally emits: SELECT pg_catalog.set_config('search_path', '', false);
-- Removed -- it empties search_path for the rest of the session/transaction,
-- which broke goose's own unqualified write to goose_db_version right after
-- this migration ran ("relation goose_db_version does not exist"). All
-- objects below are schema-qualified (public.xxx) anyway, so dropping this
-- line changes nothing about what gets created.
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_table_access_method = heap;

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
    CONSTRAINT packs_pack_type_check CHECK ((pack_type = ANY (ARRAY['bot'::text, 'server'::text, 'emoji'::text])))
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
-- Name: entity_vote_redeem_logs entity_vote_redeem_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_vote_redeem_logs
    ADD CONSTRAINT entity_vote_redeem_logs_pkey PRIMARY KEY (id);


--
-- Name: internal_user_cache__discord internal_user_cache__discord_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.internal_user_cache__discord
    ADD CONSTRAINT internal_user_cache__discord_pkey PRIMARY KEY (id);


--
-- Name: staff_positions name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staff_positions
    ADD CONSTRAINT name_unique UNIQUE (name);


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
-- Name: user_notifications poppypaw_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notifications
    ADD CONSTRAINT poppypaw_pkey PRIMARY KEY (itag);


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
-- Name: staffpanel__authchain staffpanel__authchain_itag_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.staffpanel__authchain
    ADD CONSTRAINT staffpanel__authchain_itag_key UNIQUE (itag);


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
-- Name: blacklisted_words word_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.blacklisted_words
    ADD CONSTRAINT word_unique UNIQUE (word);


--
-- Name: bots_short_fts_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bots_short_fts_idx ON public.bots USING gin (to_tsvector('english'::regconfig, short));


--
-- Name: positions_card; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX positions_card ON public.staff_members USING btree (cardinality(positions));


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
-- Name: bots bots_team_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_team_owner_fkey FOREIGN KEY (team_owner) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: bots bots_vanity_ref_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--
-- Deferred out of this migration -- public.vanity doesn't exist yet at
-- this point in the chain (it's created later, in the ported
-- vanityv2.sql migration). Added back in
-- 20260801004500_add_deferred_foreign_keys.sql, which runs after every
-- historical port.
--


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
-- Deferred -- see bots_vanity_ref_fkey note above; added back in
-- 20260801004500_add_deferred_foreign_keys.sql.
--


--
-- Name: shop_holds shop_holds_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.shop_holds
    ADD CONSTRAINT shop_holds_item_fkey FOREIGN KEY (item) REFERENCES public.shop_items(id) ON UPDATE CASCADE ON DELETE RESTRICT;


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
-- Deferred -- see bots_vanity_ref_fkey note above; added back in
-- 20260801004500_add_deferred_foreign_keys.sql.
--


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
-- Deferred out of this migration -- public.webhooks doesn't exist yet at
-- this point in the chain (it's created later, in the ported
-- webhooksdbv2.sql migration). Added back in
-- 20260801004500_add_deferred_foreign_keys.sql, which runs after every
-- historical port.
--


--
-- PostgreSQL database dump complete
--


-- +goose StatementEnd

-- +goose Down
-- Foundational tables predating all tracking -- no rollback written.
SELECT 1;
