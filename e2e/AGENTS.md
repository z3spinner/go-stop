# AGENTS.md — `e2e/` (Playwright end-to-end tests)

Browser tests that drive the **fully built app** (SvelteKit SPA served by the Go
server), exercising real flows end to end: post a ride, search, express/accept
interest, stats, etc.

## Running

`make test-e2e` (from repo root) builds the frontend then runs `playwright test`.
Config: `/playwright.config.js`.

- `webServer.command` is `npm run build && go run .` and serves on
  **http://localhost:8080** (`baseURL`). `reuseExistingServer: true` means an
  already-running stack is used if present — otherwise Playwright boots one.
- Requires **Postgres + VAPID env** available to that Go server (e.g.
  `docker compose up -d db migrations` first, or a running devstack).
- Browser timezone is pinned to `Europe/Paris`.

## Conventions

- Specs are CommonJS (`require('@playwright/test')`), `// @ts-check` at the top.
  One file today: `gostop.spec.js`, organized into numbered scenario sections.
- **Profile/locale are seeded via `localStorage`** in `page.addInitScript`, using
  the app's **raw (unquoted) string** format: `user_name`, `user_phone`, and
  `lang` (e.g. `'fr'`). Match this format — the app's persisted stores and the
  Paraglide locale strategy both read raw strings, not JSON.
- Tests assert against the **French UI** and expect the heading/title
  `"Go Stop Saillans!"` — i.e. they assume the server runs with French as the
  active locale and `SITE_NAME="Go Stop Saillans!"`. If you change copy, locale
  defaults, or `SITE_NAME`, these assertions move too.

> ⚠️ The Playwright `webServer` command sets no `SITE_NAME`/`SERVICE_TZ` itself, so
> these tests depend on those coming from the environment (a local `.env` /
> already-running stack). That coupling is fragile — see the root `AGENTS.md`
> "Known issues" list. When adding tests, prefer asserting on stable
> structure/roles over hard-coded community-specific copy where you can.
