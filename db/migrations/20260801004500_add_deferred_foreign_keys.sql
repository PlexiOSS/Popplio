-- +goose Up
-- +goose StatementBegin
-- Four FK constraints that logically belong to the genesis foundational
-- schema (20260731235950_genesis_foundational_tables.sql) but had to be
-- deferred to here: they reference public.vanity and public.webhooks,
-- neither of which exist yet at genesis time -- both are created by
-- ported historical migrations (vanityv2.sql, webhooksdbv2.sql) that run
-- later in the chain. This migration is dated after every historical
-- port so all four tables are guaranteed to exist by the time it runs.
ALTER TABLE ONLY public.bots
    ADD CONSTRAINT bots_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.servers
    ADD CONSTRAINT servers_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_vanity_ref_fkey FOREIGN KEY (vanity_ref) REFERENCES public.vanity(itag);

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON UPDATE CASCADE ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ONLY public.webhook_logs DROP CONSTRAINT IF EXISTS webhook_logs_webhook_id_fkey;
ALTER TABLE ONLY public.teams DROP CONSTRAINT IF EXISTS teams_vanity_ref_fkey;
ALTER TABLE ONLY public.servers DROP CONSTRAINT IF EXISTS servers_vanity_ref_fkey;
ALTER TABLE ONLY public.bots DROP CONSTRAINT IF EXISTS bots_vanity_ref_fkey;
-- +goose StatementEnd
