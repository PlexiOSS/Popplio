-- +goose Up
-- +goose StatementBegin
CREATE TABLE entity_page_styles (
    target_type text NOT NULL,
    target_id text NOT NULL,
    gradient text,
    accent text,
    image_version text,
    pending_image_version text,
    pending_since timestamp with time zone,
    rejected_reason text,
    reviewed_by text,
    reviewed_at timestamp with time zone,
    dim smallint DEFAULT 55 NOT NULL,
    blur smallint DEFAULT 0 NOT NULL,
    image_position text DEFAULT 'center' NOT NULL,
    fixed boolean DEFAULT false NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT entity_page_styles_pkey PRIMARY KEY (target_type, target_id),
    CONSTRAINT entity_page_styles_target_type_check CHECK (target_type IN ('bot', 'server')),
    CONSTRAINT entity_page_styles_dim_check CHECK (dim BETWEEN 30 AND 90),
    CONSTRAINT entity_page_styles_blur_check CHECK (blur BETWEEN 0 AND 24),
    CONSTRAINT entity_page_styles_position_check CHECK (image_position IN ('top', 'center', 'bottom')),
    CONSTRAINT entity_page_styles_accent_check CHECK (accent IS NULL OR accent ~ '^#[0-9a-f]{6}$')
);

CREATE INDEX entity_page_styles_pending_idx ON entity_page_styles (pending_since) WHERE pending_image_version IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE entity_page_styles;
-- +goose StatementEnd
