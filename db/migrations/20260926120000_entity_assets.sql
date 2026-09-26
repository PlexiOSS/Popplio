-- Content versions for user-uploaded CDN assets (team avatars, bot/server/team
-- banners). The bytes live in the frontend's S3 bucket under a fixed per-entity
-- key; this table records a hash of the current upload so the frontend can
-- render `/cdn/...?v=<version>` URLs that are safe to cache forever, and so a
-- re-upload is picked up immediately instead of after a cache window.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_assets (
    target_type text NOT NULL,
    target_id text NOT NULL,
    kind text NOT NULL,
    version text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT entity_assets_target_type_check CHECK (target_type = ANY (ARRAY['bot'::text, 'server'::text, 'team'::text])),
    CONSTRAINT entity_assets_kind_check CHECK (kind = ANY (ARRAY['avatar'::text, 'banner'::text]))
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY entity_assets
    ADD CONSTRAINT entity_assets_pkey PRIMARY KEY (target_type, target_id, kind);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entity_assets;
-- +goose StatementEnd
