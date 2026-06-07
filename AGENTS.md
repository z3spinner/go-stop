# AGENTS.md — Go-Stop (repository root)

Lightweight local ride-sharing notice board. A Go (Gin) API + PostgreSQL backend
serving a compiled SvelteKit SPA. No accounts; phone number is the only identity
and the (unverified) delete credential. AGPL-3.0.

> Nested `AGENTS.md` files give detailed, location-specific conventions. Read the
> one closest to the code you're touching:
> - `internal/AGENTS.md` — clean-architecture rules & layer wiring
> - `internal/usecase/AGENTS.md` — use-case pattern & unit tests
> - `internal/boundaries/AGENTS.md` — Gin handlers, OpenAPI annotations, ports
> - `internal/infrastructure/postgres/AGENTS.md` — sqlc, migrations, repos
> - `frontend/AGENTS.md` — SvelteKit, shadcn-svelte, generated API client, i18n
> - `e2e/AGENTS.md` — Playwright end-to-end tests

## Architecture (Clean Architecture — dependencies point inward only)

```
domain ← usecase ← boundaries ← infrastructure          (Go: ./internal)
                        ▲
                  main.go (composition root: builds repos → use cases → handlers → routes)

web/build  ← frontend/  (SvelteKit SPA, built and served as static files by the Go server)
```

The Go module is `github.com/z3spinner/go-stop`. `main.go` is the only wiring
point: it constructs concrete infrastructure, injects it into use cases, hands
those to handlers, and registers routes under `/api`. Any non-`/api` path falls
back to `web/build/index.html` (SPA routing), with per-ride Open Graph tags
injected server-side for link previews (`og.go`).

## Repo map

| Path | What |
|---|---|
| `main.go`, `og.go`, `main_serve_test.go` | Composition root, SPA+OG handler, serve tests |
| `internal/domain` | Pure entities/value objects (no deps, no tags) |
| `internal/usecase` | Business logic; one `NewXxx`/`Execute` struct per file |
| `internal/boundaries` | Gin handlers + repository/notifier **interfaces** (ports) |
| `internal/infrastructure` | postgres (pgx + sqlc), vapid, webpush — the adapters |
| `internal/version` | Build SHA, injected at build time (do not commit `build.go`) |
| `cmd/migratedb` | golang-migrate CLI (`up`/`down`/`force`/`drop`/`version`) |
| `frontend/` | SvelteKit + TypeScript + shadcn-svelte; builds to `../web/build` |
| `e2e/` | Playwright specs (run against the built app on :8080) |
| `docs/` | **Generated** OpenAPI (`docs.go`, `swagger.json/yaml`) + design notes |
| `scripts/seed.sh` | Seeds the dev DB through the running app's API |
| `web/build` | Build artifact (gitignored) |

## Commands (see `Makefile`)

| Task | Command |
|---|---|
| Run dev stack (DB + API + Vite) | `docker compose up --build` → app on :5173 (proxies `/api`→:8080) |
| Run Go + Vite locally (no Docker) | `make dev` |
| Unit tests (fast, no DB) | `make test-unit` |
| Integration tests (isolated stack) | `make test-integration` (alias: `make test`) |
| E2E tests (isolated stack) | `make test-e2e` |
| Unit + integration + e2e (stacks in parallel) | `make test-all` |
| Lint Go (golangci-lint) | `make lint` (`make lint-install` once to install the pinned version) |
| Auto-fix Go formatting | `make fmt` |
| Regenerate SQL code | `make sqlc` (after editing `*.sql`) |
| Regenerate OpenAPI spec | `make swagger` (after changing handler annotations) |
| Regenerate OpenAPI **+ frontend client** | `make api-generate` |
| Build frontend | `make build-web` |
| Seed dev DB | `make seed` |

**Integration and e2e run in their own throwaway docker compose projects**
(`gostop-itest` / `gostop-e2e`, see `docker-compose.itest.yml` /
`docker-compose.e2e.yml`) with **no published host ports** — so they spin up
their own Postgres (and, for e2e, the production app image + Playwright), never
touch the devstack's DB or ports, and run in parallel with the devstack and each
other. No need to start Postgres yourself, and `make test` no longer truncates
the dev database.

## Code generation pipeline — keep these in sync

Three generated artifacts are committed and must be regenerated (never
hand-edited) when their source changes:

1. **sqlc** — edit `internal/infrastructure/postgres/sqlc/queries/sql/*.sql`, then
   `make sqlc`. Regenerates `.../queries/*.sql.go`.
2. **OpenAPI** — change swaggo annotations on handlers, then `make swagger`.
   Regenerates `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.
3. **Frontend API client** — `make api-generate` runs swagger then orval, which
   regenerates `frontend/src/lib/api/generated/go-stop-api.ts` from the spec.

A handler/endpoint change typically touches all three: update SQL/use case →
`make sqlc`, update annotations → `make api-generate`, commit the generated files.

## Configuration (env vars)

`DATABASE_URL` (required), `PORT` (default 8080), `SITE_NAME` (heading; default
`Go-Stop`), `SERVICE_TZ` (IANA tz used to interpret time-only searches; default
UTC, `Europe/Paris` in compose), `RIDE_GRACE_MINUTES` (default 60),
`RETURN_DELAY_HOURS` (default 2), `GIN_MODE` (`release` in prod), and the
`VAPID_*` Web Push keys (auto-generated and persisted to the DB on first boot if
unset — see `internal/infrastructure/vapid`). Copy `.env.example` → `.env` for
local work.

## Deployment (Scalingo)

`Procfile`: `web: migratedb up && go-stop` — migrations run at boot, before the
server starts (postdeploy is **not** used for migrations; boot-time DB reads
depend on this ordering). `.buildpacks` chains the Node then Go buildpacks;
`bin/go-pre-compile` injects the git SHA into `internal/version/build.go`.
`scalingo.json` defines the one-click deploy form. The production image
(`Dockerfile`) is a 3-stage build: Vite build → Go build → alpine runtime.

## Licensing & file headers

The project is **AGPL-3.0-or-later** (`LICENSE`, `NOTICE`). **Every new
hand-written source file must start with an SPDX header.**

Go / TypeScript / JavaScript (`.go`, `.ts`, `.js`):

```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later
```

Svelte (`.svelte`) — leading HTML comment:

```svelte
<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->
```

In Go, the header goes above everything; for build-tagged files keep the order
`SPDX → blank line → //go:build → blank line → package`.

**Do NOT** add headers to generated or vendored code — it isn't ours to mark:
sqlc queries, `docs/docs.go`, `internal/version/build.go`,
`frontend/src/lib/api/generated/`, the shadcn-svelte `frontend/src/lib/components/ui/`
primitives, and Paraglide output. (Update the copyright holder for genuinely new
contributors as appropriate.)

## ⚠️ Known issues to fix (these are NOT conventions — do not imitate)

- The many loose `*.png` screenshots at the repo root are clutter (already
  gitignored via `*.png`, but messy to have on disk). _(Fixed: a 35 MB compiled
  `go-stop` binary that had been committed is now removed and `/go-stop`,
  `/migratedb`, `/backfill-matches` are gitignored.)_
- The migration numbering on `main` jumps 006 → 008: phone-at-rest encryption and
  its migration `007_widen_phone_columns` live only on the `feature/phone-encryption`
  branch. golang-migrate tolerates the gap (it tracks versions by name), so this is
  harmless — just **do not reuse 007**. _(Fixed: the unimplemented
  `PHONE_ENCRYPTION_KEY` env var that had been advertised in `scalingo.json` is now
  removed, since no Go code on `main` reads it.)_
_(All earlier README/config call-outs have been fixed: the committed binary, the
dead `PHONE_ENCRYPTION_KEY` env var, and the stale `README.md` dev/test
instructions — which now point at :5173/Vite, `make dev`, and the real `make test`
flow instead of the removed `docker-compose.test.yml`.)_
