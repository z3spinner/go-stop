# AGENTS.md — `e2e/` (Playwright end-to-end tests)

Browser tests that drive the **fully built app** (SvelteKit SPA served by the Go
server), exercising real flows end to end: post a ride, search, express/accept
interest, stats, etc.

## Running

`make test-e2e` (from repo root) brings up the isolated **`gostop-e2e`** docker
compose project (`/docker-compose.e2e.yml`) and runs the suite against it, then
tears it down. Config: `/playwright.config.js`.

- The stack has **no published host ports**, so it coexists with the devstack and
  the integration stack (run everything with `make test-all`).
- `app` is the **production** image (self-contained SPA + Go API) with
  `SITE_NAME="Go Stop Saillans!"` and `SERVICE_TZ=Europe/Paris` set by the compose
  file — no host build, no external env coupling.
- The Playwright runner **shares the app's network namespace**
  (`network_mode: service:app`) and drives it at `http://localhost:8080`
  (`E2E_BASE_URL`). localhost matters: Chromium auto-upgrades non-localhost http
  navigations to https, which fails against the TLS-less app
  (`ERR_SSL_PROTOCOL_ERROR`).
- `baseURL` and the spec's `BASE` honor `E2E_BASE_URL`. With it **unset**, a bare
  local `npx playwright test` falls back to building + booting its own Go server
  on :8080 (`webServer`, `reuseExistingServer: true`) — handy for a quick local
  run, but it then depends on `SITE_NAME`/`SERVICE_TZ` from your environment.
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

> The canonical `make test-e2e` path sets `SITE_NAME`/`SERVICE_TZ` in the compose
> file, so its assertions are self-contained. Only the fallback bare-`npx`
> `webServer` path depends on those coming from your environment. When adding
> tests, still prefer asserting on stable structure/roles over hard-coded
> community-specific copy where you can.
