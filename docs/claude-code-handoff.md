# Claude Code Handoff — Go-Stop

This document is a prompt for Claude Code to scaffold the Go-Stop project from scratch based on the architecture and design decisions made during planning.

---

## Your Task

Scaffold the entire Go-Stop project into this repository based on the design documents in the `/docs` folder. Read them carefully before writing any code.

## Key Documents

- `docs/requirements.md` — product requirements and user flows
- `docs/design.md` — architecture, project structure, domain model, use cases, boundaries, and API
- `docs/data-model.md` — PostgreSQL schema, indexes, matching query, expiry

## What to Scaffold

1. `go.mod` — module `github.com/z3spinner/go-stop`, Go 1.22+
2. `main.go` — wire everything together, start Gin server, run expiry cron
3. `Procfile` — for Scalingo deployment
4. `scalingo.json` — one-click deploy manifest with PostgreSQL add-on and VAPID env vars
5. `README.md` — with Deploy on Scalingo button pointing to `https://github.com/z3spinner/go-stop`
6. `internal/domain/` — all domain types
7. `internal/usecase/` — all use cases with repository and notifier interfaces injected
8. `internal/boundaries/handler/` — Gin HTTP handlers (thin, delegate to use cases)
9. `internal/boundaries/repository/` — repository interfaces
10. `internal/boundaries/notification/` — notifier interface
11. `internal/infrastructure/postgres/` — PostgreSQL implementations using `pgx` or `database/sql`
12. `internal/infrastructure/webpush/` — Web Push implementation using `github.com/SherClockHolmes/webpush-go`
13. `web/index.html` — simple homepage with "I'm driving" and "I need a ride" entry points
14. Database migration file(s) in `db/migrations/`

## Architecture Rules

- Follow Clean Architecture strictly — dependencies point inward only
- Domain layer has zero external dependencies
- Use cases depend only on repository and notifier interfaces
- Gin lives only in the boundaries/handler layer
- PostgreSQL lives only in infrastructure/postgres
- Web Push lives only in infrastructure/webpush

## Notes

- Phone number is used as lightweight auth for deletions (no accounts)
- Flexibility is stored in minutes as an integer, with presets: Exact=0, Approximate=30, Flexible=60
- Expiry cron should run every hour
- VAPID keys are provided via environment variables: VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, VAPID_EMAIL
- DATABASE_URL and PORT are set automatically by Scalingo
