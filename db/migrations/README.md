# Migrations

Versioned schema changes, applied with [goose](https://github.com/pressly/goose) via `popplio/cmd/migrate`. This replaces the old `exp/*.sql`-files-run-by-hand-via-psql workflow (that folder has since been deleted -- everything in it was ported here first; see the git history around 2026-08-28 if you need the old originals). Lives under `popplio/db/` alongside `../queries` (sqlc query definitions) and the generated `popplio/db` package, since both draw from the same schema.

## Workflow

```sh
go run ./cmd/migrate create add_some_column   # scaffold a new migration
go run ./cmd/migrate status                   # see what's applied / pending
go run ./cmd/migrate up                       # apply every pending migration
```

`create` writes a timestamp-prefixed `.sql` file here with `-- +goose Up` / `-- +goose Down` sections fill in the real SQL for both directions before committing. `up`/`status` connect using the same `config.yaml` (`meta.postgres_url`) the rest of Popplio reads, so whichever environment's config you're running against is the one that gets migrated **double check `config.yaml` before running `up`**, especially since it's common for a local checkout's config to already point at a shared staging/prod database rather than something local.

Applied versions are tracked in a `goose_db_version` table goose creates automatically on first run no manual bookkeeping needed.

This is deliberately a manual step, not wired into `make`/deploy run it yourself before restarting the service, so a migration is never something that happens silently as a side effect of a deploy.
