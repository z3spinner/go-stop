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

- Go 1.22+
- PostgreSQL 14+

## Local setup (with Docker — recommended)

```bash
cp .env.example .env
# Optionally edit .env to set SITE_NAME for your community.
# VAPID keys are generated automatically on first boot if left blank.

docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080). The database is created and migrated automatically.

The `app` service uses `reflex` for hot-reload: any change to a `.go` file triggers an automatic rebuild and restart. Changes to `web/` (HTML, CSS, JS) are served live from the host volume with no restart needed — just refresh your browser.

## Local setup (manual)

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/gostop?sslmode=disable"
# VAPID keys are optional — generated and stored in the DB on first boot if unset.
export PORT=8080

psql $DATABASE_URL < db/migrations/001_create_tables.sql
go run .
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

## Integration tests

Requires a running PostgreSQL database:

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml up db -d
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  go test -tags integration ./...
docker compose -f docker-compose.yml -f docker-compose.test.yml down
```

The test override (`docker-compose.test.yml`) replaces the persistent volume with `tmpfs` so data lives only in RAM and is discarded when the container stops. The plain `docker compose up` devstack keeps its data across restarts.

## License

[GNU Affero General Public License v3.0](LICENSE) — any modifications must be made available under the same license, including when the software is run as a network service.
