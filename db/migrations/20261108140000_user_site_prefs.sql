-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_site_prefs (
    user_id text NOT NULL,
    prefs jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_site_prefs_pkey PRIMARY KEY (user_id),
    CONSTRAINT user_site_prefs_user_fkey FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT user_site_prefs_object_check CHECK (jsonb_typeof(prefs) = 'object')
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_site_prefs;
-- +goose StatementEnd
