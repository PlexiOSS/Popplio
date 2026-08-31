-- +goose Up
-- +goose StatementBegin
-- User-submitted Discord server templates (discord.com/template/<code> --
-- a shareable snapshot of a server's channel/role/permission structure).
-- Same trust model as packs: no staff review queue, owner creates/deletes
-- directly. name/short come from Discord's own public, unauthenticated
-- template-metadata endpoint at submission time, not editable after --
-- keeps the submission flow to "paste a code, pick tags."
CREATE TABLE public.server_templates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL PRIMARY KEY,
    code text NOT NULL,
    name text NOT NULL,
    short text NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    nsfw boolean DEFAULT false NOT NULL,
    owner text NOT NULL,
    usage_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX server_templates_code_key ON public.server_templates (code);
CREATE INDEX server_templates_owner_idx ON public.server_templates (owner);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public.server_templates;
-- +goose StatementEnd
