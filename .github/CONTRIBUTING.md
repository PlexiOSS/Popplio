# Contributing to Popplio

## Branching

`develop` is the integration branch every feature/fix PR targets it, not
`staging` or `production` directly. Those two are promoted separately: a
`develop → staging` merge and the `staging → production` promotion (opened
automatically by the **Promote staging to production** workflow, but never
auto-merged a human still reviews and merges it) are how changes actually
reach production, not something a contributor PR does directly.

Branch names follow `<type>/<short-description>`, matching what's already
in the repo's history:

- `feat/...` a new feature or capability
- `fix/...` a bug fix
- `chore/...` everything else (deps, cleanup, tooling, docs)

For example: `feat/team-stats`, `fix/infernoplex-invite-ssrf`.

## Opening a pull request

1. Branch off `develop`.
2. Keep the PR scoped to one change easier to review, easier to revert if
   something's wrong with it.
3. Open the PR against `develop`.
4. Fill in what changed and why. If it touches the database, say so and
   include the migration if one's needed (see below).

`develop` is a protected branch: PRs need required status checks passing
and a review before they can merge. Force-pushing or bypassing the checks
isn't available to contributors.

## Required checks

Every PR runs:

- **Go Build & Vet** (`go.yml`) `go build ./...` and `go vet ./...` for
  both `popplio` and `cmd/kitehelper` (it has its own `go.mod`).
- **CodeQL Advanced** (`codeql.yml`) static analysis across Go, Python,
  and the workflow files themselves.
- **Kitescratch Tests** (`tests.yml`) `cmd/kitehelper`'s own test suite,
  including the struct-vs-schema (`@ci`) validation that catches a column
  drifting out of sync with its Go struct.
- **Socket Security** dependency supply-chain scanning on anything the
  PR adds or changes in `go.mod`/`go.sum`.

All of them need to pass before a PR can merge into `develop`. If a check
fails, fix the underlying issue don't route around it.

## Database changes

Schema changes are versioned [goose](https://github.com/pressly/goose)
migrations under `db/migrations/`, applied with `go run ./cmd/migrate up`.
See `db/migrations/README.md` for the workflow. Migrations are **not** run
automatically by CI or on startup they're a deliberate, manual step, so if
your PR needs one, say so explicitly in the PR description: whoever merges
it needs to run `up` separately before restarting the service.

## Code style

- Run `gofmt` before committing. CI doesn't currently enforce this
  repo-wide (there's existing drift), but new code should be clean.
- Match the file's own conventions before reaching for a personal
  preference this codebase has strong, deliberate conventions
  (`arcadia/CONFORMANCE.md` is a good example of how seriously prior
  behavior is documented and preserved).
- Don't add a config value split across environments there's only one
  deployment now (see the Changelog's `[Unreleased]` entry on this if
  you're wondering why that used to exist).

## Questions

Open an issue or ask in the project's Discord before starting on anything
large better to align on approach before writing the PR than to redo it
after review.
