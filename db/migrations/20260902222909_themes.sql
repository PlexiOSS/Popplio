-- +goose Up
-- +goose StatementBegin
CREATE TABLE themes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    owner text NOT NULL,
    name text NOT NULL,
    primary_color text NOT NULL,
    secondary_color text NOT NULL,
    tags text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE ONLY themes ADD CONSTRAINT themes_pkey PRIMARY KEY (id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX themes_owner_idx ON themes USING btree (owner);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE themes;
-- +goose StatementEnd
