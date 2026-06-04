# Go-Stop

A lightweight local ride-sharing notice board. Drivers post one-time trips; searchers browse or post waiting requests. Matches trigger instant push notifications. Direct contact via phone number — no accounts, no in-app messaging.

[![Deploy to Scalingo](https://cdn.scalingo.com/deploy/button.svg)](https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop)

## How it works

- **Drivers** post a ride with origin, destination, date, departure time, and flexibility window
- **Searchers** browse by origin/destination or post a waiting request
- Both parties receive a **push notification** when a match is found
- Contact is made directly via the displayed phone number

## ⚠️ Security model

This app uses phone number as a lightweight delete credential with **no verification**. Anyone who knows a ride's phone number can delete it. This is an intentional design tradeoff — frictionless entry over strong authentication. Do not use for sensitive or high-stakes scenarios.

## Requirements

- Go 1.25+
- Node 22+ (for the SvelteKit frontend)
- PostgreSQL 14+

## Local setup (with Docker — recommended)

```bash
cp .env.example .env
# Optionally edit .env to set SITE_NAME for your community.
# VAPID keys are generated automatically on first boot if left blank.

docker compose up --build
```

Open **[http://localhost:5173](http://localhost:5173)** — the SvelteKit/Vite dev
server, which proxies `/api` to the Go server on :8080. The database is created
and migrated automatically.

The devstack runs four services with hot-reload on both sides:

- **`frontend`** — Vite dev server on :5173; edits under `frontend/` reload live.
- **`app`** — the Go API on :8080, using `reflex`: any `.go` change triggers a
  rebuild and restart.
- **`db`** + **`migrations`** — Postgres (migrated on startup). The dev database
  is **tmpfs (in-memory): data is discarded when the stack stops.**

After changing frontend dependencies, re-run `docker compose up --build` (or
`docker compose down -v` to reset the cached `node_modules` volume).

## Local setup (manual)

Run the Go API and the Vite dev server together with hot-reload:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/gostop?sslmode=disable"
# VAPID keys are optional — generated and stored in the DB on first boot if unset.

npm ci --prefix frontend    # install frontend deps (first run only)
go run ./cmd/migratedb up   # apply database migrations
make dev                    # Go on :8080 + Vite on :5173 (proxying /api) — open :5173
```

To instead run a single Go server that serves the built SPA (as in production),
build the frontend first so `web/build` exists:

```bash
make build-web   # npm ci + vite build → web/build
go run ./cmd/migratedb up
go run .          # serves the SPA + API on :8080
```

## Deployment (Scalingo)

Click the button above — it's genuinely one click. No fields are required: the
app generates its own Web Push (VAPID) keypair on first boot and stores it in the
database, so there's nothing to paste in.

You can optionally customise these from the Scalingo deploy form (all have sensible defaults):

| Variable | Description |
|---|---|
| `SITE_NAME` | **Your community's name** shown as the site heading (e.g. `Stop Nyons`, `Covoiturage Drôme`) |
| `SERVICE_TZ` | Community timezone, used to interpret time-only searches (e.g. `Europe/Paris`) |

## Architecture

Clean Architecture — dependencies point inward only.

```
domain ← usecase ← boundaries ← infrastructure
```

| Layer | Contents |
|---|---|
| `internal/domain` | Entities: Ride, Request, Subscription, Message |
| `internal/usecase` | Business logic with injected interfaces |
| `internal/boundaries` | Gin handlers + repository/notifier interfaces |
| `internal/infrastructure` | PostgreSQL (pgx) + Web Push (VAPID) |
| `web/` | Static SPA served by Go |

## Tests

Unit tests (no database):

```bash
make test-unit   # go test ./internal/usecase/...
```

Integration tests need a running PostgreSQL. Start just the `db` and `migrations`
services, then run the suite:

```bash
docker compose up -d db migrations
make test   # go test -tags integration -count=1 -p 1 ./... against localhost:5432
docker compose down
```

`make test` points `TEST_DATABASE_URL` at the compose database. That database is
tmpfs-backed, so its contents live only in RAM and are discarded when the stack
stops — no separate test override is needed.

End-to-end (Playwright) tests build the SPA and run it against the Go server:

```bash
make test-e2e
```

## License

[GNU Affero General Public License v3.0](LICENSE) — any modifications must be made available under the same license, including when the software is run as a network service.
