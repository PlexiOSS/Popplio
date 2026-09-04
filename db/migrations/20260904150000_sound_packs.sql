-- +goose Up
-- +goose StatementBegin
ALTER TABLE packs DROP CONSTRAINT packs_pack_type_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE packs ADD CONSTRAINT packs_pack_type_check
    CHECK (pack_type = ANY (ARRAY['bot'::text, 'server'::text, 'emoji'::text, 'sticker'::text, 'sound'::text]));
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE pack_sounds (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pack_url text NOT NULL,
    name text NOT NULL,
    duration_ms integer DEFAULT 0 NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    downloads integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY pack_sounds ADD CONSTRAINT pack_sounds_pkey PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY pack_sounds
    ADD CONSTRAINT pack_sounds_pack_url_fkey FOREIGN KEY (pack_url) REFERENCES packs(url) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX pack_sounds_pack_url_idx ON pack_sounds USING btree (pack_url);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE pack_sounds;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE packs DROP CONSTRAINT packs_pack_type_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE packs ADD CONSTRAINT packs_pack_type_check
    CHECK (pack_type = ANY (ARRAY['bot'::text, 'server'::text, 'emoji'::text, 'sticker'::text]));
-- +goose StatementEnd
