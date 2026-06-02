# Frontend Refactor: SvelteKit + TypeScript + Vite + shadcn-svelte

**Date:** 2026-06-02  
**Branch:** `refactor/frontend`  
**Status:** Approved — ready for implementation plan

---

## Summary

Replace the monolithic `web/js/app.js` (2489 lines, vanilla JS) with a SvelteKit SPA using TypeScript, Vite, shadcn-svelte, and Paraglide for i18n. The Go backend and API are unchanged. All 32 existing Playwright e2e tests must pass before the branch merges.

---

## 1. Tech Stack

| Concern | Choice | Rationale |
|---|---|---|
| Framework | SvelteKit + `@sveltejs/adapter-static` | File-based routing, static output, full shadcn-svelte support |
| Language | TypeScript | Catch missing API fields and translation keys at build time |
| Build | Vite (built into SvelteKit) | Content-hashed filenames replace Go's `{{.Version}}` cache busting |
| UI primitives | shadcn-svelte | Pre-built accessible components, works on top of Bits UI |
| i18n | Paraglide JS | Type-safe, compile-time, zero runtime overhead; missing keys = build error |
| Unit tests | Vitest + @testing-library/svelte | Standard SvelteKit testing stack |
| E2e tests | Playwright (existing) | 32-test suite preserved; selectors updated where needed |

---

## 2. Project Structure

```
go-stop/
├── frontend/                       ← SvelteKit project root
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api.ts              ← typed API client
│   │   │   ├── stores.ts           ← profile, lang, pushState, lastSearch
│   │   │   └── pwa.ts              ← push subscription, A2HS, standalone detection, polling
│   │   ├── routes/
│   │   │   ├── +layout.svelte      ← TopBar, A2HS banner, poll toast mount
│   │   │   ├── +page.svelte        ← home
│   │   │   ├── post-ride/+page.svelte
│   │   │   ├── search/+page.svelte
│   │   │   ├── my-rides/+page.svelte
│   │   │   ├── my-searches/+page.svelte
│   │   │   ├── my-alerts/+page.svelte     ← redirects to /my-searches
│   │   │   ├── my-requests/+page.svelte   ← redirects to /my-searches
│   │   │   ├── me/+page.svelte
│   │   │   ├── stats/+page.svelte
│   │   │   └── interests/[id]/+page.svelte
│   │   ├── messages/               ← Paraglide message files
│   │   │   ├── en.json
│   │   │   ├── fr.json
│   │   │   ├── es.json
│   │   │   ├── it.json
│   │   │   ├── de.json
│   │   │   └── nl.json
│   │   └── app.html                ← SvelteKit HTML shell (replaces web/index.html)
│   ├── static/                     ← copied verbatim into build output
│   │   ├── sw.js
│   │   ├── manifest.json
│   │   ├── logo.svg
│   │   ├── icon-192.png
│   │   ├── icon-512.png
│   │   ├── icon-maskable-192.png
│   │   ├── icon-maskable-512.png
│   │   └── apple-touch-icon.png
│   ├── svelte.config.ts
│   ├── vite.config.ts
│   └── package.json
├── web/
│   └── build/                      ← gitignored; adapter-static output dir
├── package.json                    ← root; delegates build to frontend/
├── .buildpacks                     ← Scalingo: Node.js buildpack then Go buildpack
├── bin/go-pre-compile              ← unchanged (version injection only)
├── internal/                       ← unchanged
└── main.go                         ← updated to serve ./web/build/
```

---

## 3. Routes

| SvelteKit route | Current handler | Notes |
|---|---|---|
| `/` | `renderHome` | Ride feed + ghost nav tiles |
| `/post-ride` | `renderPostRide` | Ride form with return-leg toggle |
| `/search` | `renderSearchRides` | `?origin&destination&departure_at&search_date&search_time` preserved |
| `/my-rides` | `renderMyRides` | Driver's own rides + seeker rows + Ping button |
| `/my-searches` | `renderMySearches` | Combined alerts + contact requests |
| `/my-alerts` | — | `goto('/my-searches')` on load |
| `/my-requests` | — | `goto('/my-searches')` on load |
| `/me` | `renderMe` | Profile name/phone editor |
| `/stats` | `renderStats` | Statistics page |
| `/interests/[id]` | `renderInterestContact` | Contact reveal page |
| `/rides/[id]` | ride detail deep link | Push notification deep link |

SvelteKit's `goto()` for client navigation; `browser` guard on all localStorage/push code.

---

## 4. Component Architecture

```
src/lib/components/
├── layout/
│   ├── TopBar.svelte          ← lang toggle, me icon, bell/A2HS hint
│   └── PageBar.svelte         ← back button + TopBar
├── rides/
│   ├── RideCard.svelte        ← public listing card
│   ├── RideForm.svelte        ← post-ride form (incl. return leg)
│   └── SeekerRow.svelte       ← driver view of matching searcher + Ping button
├── alerts/
│   ├── AlertCard.svelte       ← saved alert with delete / see-matches
│   └── AlertForm.svelte       ← post-request form
├── requests/
│   └── RequestCard.svelte     ← contact request card (pending / accepted)
├── notifications/
│   ├── BellButton.svelte      ← bell icon, state-aware (enabled/disabled/A2HS)
│   ├── NotifModal.svelte      ← enable / skip / denied dialog
│   ├── A2HSModal.svelte       ← step-by-step install instructions + iOS version note
│   ├── A2HSBanner.svelte      ← dismissible home-page banner (iOS non-standalone)
│   └── PollToast.svelte       ← in-app notification toast with View button
└── ui/                        ← shadcn-svelte: Button, Card, Input, Select,
                                  Badge, Dialog, Toast, Label
```

---

## 5. Shared State (`src/lib/stores.ts`)

```typescript
export const profile   = persisted<{ name: string; phone: string }>('profile', { name: '', phone: '' });
export const lang      = persisted<string>('lang', 'fr');
export const pushState = writable<'default' | 'granted' | 'denied' | 'subscribed'>('default');
export const lastSearch = persisted<{ origin: string; destination: string }>('lastSearch', { origin: '', destination: '' });
```

`persisted()` from `@svelte-persisted-store` (thin wrapper over localStorage).

---

## 6. API Layer (`src/lib/api.ts`)

Typed domain types mirror Go JSON responses. Single `apiFetch` helper throws on non-2xx. Namespaced by resource:

```typescript
export const api = {
  rides:         { list, get, post, del, listInterests, listMatchingRequests },
  requests:      { list, post, del },
  interests:     { express, accept, getContact, listMine },
  subscriptions: { upsert, remove },
  notifications: { list },
  config:        { get },
  stats:         { get },
  vapid:         { getPublicKey },
}
```

In dev, `vite.config.ts` proxies `/api/*` → `http://localhost:8080`. In production the SvelteKit static build and Go binary share the same origin.

---

## 7. PWA / Push Notifications (`src/lib/pwa.ts`)

All push logic extracted from `app.js` into a typed module:

- `trySubscribePush(phone)` — subscribe and POST to `/api/subscriptions`
- `maybeShowStandaloneNotifPrompt()` — fires once on first standalone launch
- `pollForNotifications()` — called on `visibilitychange`, shows `PollToast`
- `updateBellState()` — detects expired subscription and silently resubscribes

The service worker (`static/sw.js`) is unchanged — it handles push events and notification clicks.

---

## 8. Build Pipeline

### Local dev

```bash
make dev   # starts Go (port 8080) + Vite dev server (port 5173) concurrently
```

`vite.config.ts` proxy:
```typescript
server: { proxy: { '/api': 'http://localhost:8080' } }
```

### Production (Scalingo)

`.buildpacks`:
```
https://github.com/Scalingo/nodejs-buildpack
https://github.com/Scalingo/go-buildpack
```

Root `package.json`:
```json
{ "scripts": { "build": "npm ci --prefix frontend && npm run build --prefix frontend" } }
```

Node.js buildpack runs `npm run build` → outputs to `web/build/`.  
Go buildpack compiles the binary.  
`bin/go-pre-compile` is unchanged (version injection only — no longer needed for cache busting but kept for `internal/version`).

### `main.go` changes

Replace individual `r.Static("/css")` / `r.Static("/js")` / `r.StaticFile(...)` calls with:

```go
r.Static("/", "./web/build")
r.NoRoute(func(c *gin.Context) {
    if strings.HasPrefix(c.Request.URL.Path, "/api") {
        c.Status(http.StatusNotFound)
        return
    }
    c.File("./web/build/index.html")
})
```

`IndexHandler` and the `{{.Version}}` template are removed — Vite handles cache busting via content-hashed filenames.

---

## 9. Testing

### Unit / component (Vitest)

- **≥1 test per component** covering render, interaction, and edge cases
- Key tests: `RideCard` renders without phone, `AlertCard` delete triggers API, `NotifModal` state machine, `BellButton` shows A2HS label on iOS
- API client: `vi.stubGlobal('fetch', ...)` for typed response and error-path coverage
- Stores: profile persistence, lang switching

### E2e (Playwright)

- Existing `e2e/gostop.spec.js` preserved (32 tests)
- Selectors updated where SvelteKit changes element IDs/classes
- `playwright.config.ts` updated to run against Vite dev server on port 5173
- `make test-e2e` target added to Makefile

---

## 10. Migration Notes

- `app.js` and `web/css/style.css` are **deleted** when the branch ships
- `web/index.html` is **replaced** by `frontend/src/app.html`
- All i18n strings are **migrated** from the JS translation objects to Paraglide `.json` message files (6 languages × ~60 keys)
- The `esc()` helper is replaced by Svelte's automatic HTML escaping
- The `formatTime()` / `formatDate()` helpers move to `src/lib/utils.ts`

---

## Acceptance Criteria

- [ ] `make dev` starts both servers; app works at `http://localhost:5173`
- [ ] `npm run build --prefix frontend` produces `web/build/` with no errors
- [ ] Go binary serves `web/build/` and all API routes work
- [ ] All 32 Playwright e2e tests pass
- [ ] Vitest unit tests pass with ≥1 test per component
- [ ] TypeScript strict mode: zero `any`, zero type errors
- [ ] Paraglide: zero missing translation keys across all 6 languages
- [ ] PWA: push notifications, A2HS flow, standalone detection all preserved
- [ ] Scalingo deployment succeeds with `.buildpacks` multi-buildpack config
