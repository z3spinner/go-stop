# Isolated, parallel test stacks — design

**Date:** 2026-06-07
**Status:** Approved

## Goal

Run the **integration** and **e2e** test suites in fully isolated stacks that:
- never touch the devstack's database or ports,
- can run in parallel with each other and with the devstack,
- are triggered reliably from the `Makefile`.

Today `make test` points `TEST_DATABASE_URL` at the devstack DB (`gostop` on
:5432) and **truncates it**, and `make test-e2e` builds on the host and targets
:8080 with `reuseExistingServer`, colliding with the devstack app.

## Approach

Each test type is its own **docker compose project** with **no published host
ports**. The test runner executes as a container *inside* each stack's private
network and reaches services by name. Nothing maps to the host, so the stacks
cannot collide with the devstack (`go-stop`: 5432/8080/5173) or each other, and
any number can run at once.

## Components

### 1. Integration stack — `docker-compose.itest.yml` (project `gostop-itest`)

- `db`: `postgres:17-alpine`, internal only, `pg_isready` healthcheck.
- `tests`: Dockerfile **`development`** target (Go 1.25). Mounts the working tree
  (`.:/app`) and a dedicated module-cache volume. `depends_on: db (healthy)`.
  Command:
  ```sh
  go run ./cmd/migratedb up && go test -tags integration -count=1 -p 1 ./...
  ```
  Env: `DATABASE_URL` and `TEST_DATABASE_URL` =
  `postgres://gostop:gostop@db:5432/gostop?sslmode=disable`.

### 2. E2e stack — `docker-compose.e2e.yml` (project `gostop-e2e`)

- `db`: `postgres:17-alpine`.
- `app`: Dockerfile **`production`** target (self-contained SPA + Go API — no
  host build, so the root-owned `.svelte-kit`/`web/build` artifact problem
  disappears). Command `sh -c "./migratedb up && ./go-stop"`. Env:
  `DATABASE_URL=…@db:5432`, `SITE_NAME="Go Stop Saillans!"`,
  `SERVICE_TZ=Europe/Paris`, `PORT=8080`. Healthcheck: `wget` on
  `http://localhost:8080/api/config`. (The server serves `./web/build` from disk
  — `main.go:153` — and self-provisions VAPID keys into the DB.)
- `tests`: `mcr.microsoft.com/playwright:v1.60.0-noble` (matches
  `@playwright/test@1.60.0`). Mounts the repo + a node_modules volume.
  `depends_on: app (healthy)`. Command `sh -c "npm ci && npx playwright test"`.
  Env: `E2E_BASE_URL=http://app:8080`.

### 3. `playwright.config.js` change

- `baseURL = process.env.E2E_BASE_URL || 'http://localhost:8080'`.
- Start the built-in `webServer` **only when `E2E_BASE_URL` is unset**, so the
  containerized stack targets the compose `app`, while a bare local
  `npx playwright test` keeps booting its own server as before.

### 4. `.dockerignore` (new)

`COPY . .` in the builder currently has no ignore file. Add one excluding
`node_modules`, `frontend/node_modules`, `frontend/.svelte-kit`, `web/build`,
`frontend/build`, `.git`, `.svelte-kit`, test/CI artifacts — shrinking the build
context and keeping root-owned host artifacts out of the e2e image build. Safe
because the Go binary serves `web/build` from disk and the frontend stage builds
it fresh inside the image.

### 5. Makefile targets

```
test-unit         host `go test ./internal/usecase/...` (no DB; unchanged)
test-integration  bring up gostop-itest, run, capture exit, always `down -v`
test-e2e          bring up gostop-e2e, run, capture exit, always `down -v`
test              alias for test-integration
test-all          test-unit, then test-integration & test-e2e in parallel
```

Containerized targets use
`up --build --abort-on-container-exit --exit-code-from tests`, then **always**
`down -v --remove-orphans`, propagating the runner's exit code so `make` fails
when tests fail. Each stack uses its own module-cache / node_modules volume
(isolation; first run downloads deps, subsequent runs are cached).

## Out of scope

- CI is unchanged — it runs the suite directly against a service Postgres, not via
  `make`. These targets are for local use; CI could adopt them later.
- The host-based `make test-e2e`/`build-web` path is replaced by the
  containerized one.

## Testing the change itself

- `make test-integration` and `make test-e2e` pass while the devstack is up, and
  leave the devstack DB (`gostop`) untouched (verify row counts before/after).
- The two run concurrently (`make test-all`) without port/DB conflicts.
