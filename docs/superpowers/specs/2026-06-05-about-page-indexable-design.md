# Design: indexable `/about` page

- **Date:** 2026-06-05
- **Status:** Approved (ready for implementation plan)
- **Author:** Zeno Kerr (with Claude)

## Problem

"About" is currently a **client-only modal** (`AboutModal.svelte`, opened by the
ⓘ icon in `TopBar`). The whole app is a client-rendered SPA
(`+layout.ts`: `ssr = false`, `prerender = false`), so there is no real page at a
crawlable URL and the About copy never appears in server HTML. Google cannot
reliably index it.

## Goal

Turn About into a real, **prerendered, Google-indexable `/about` page**, while the
live SPA still renders About in all six languages (client-side i18n, switchable
like the rest of the app).

## Approved decisions

1. **Replace the modal** with a `/about` route. The ⓘ icon navigates to `/about`;
   the modal is removed.
2. **Prerender** `/about` to static HTML (`web/build/about.html`) — reliably
   indexable, no dependency on Googlebot executing JS.
3. **Full app layout** — `/about` renders through the existing TopBar/footer
   chrome (pixel-consistent with the app).
4. **Crawler snapshot language = French** (`baseLocale`). Live users still get all
   six languages via client hydration + reactive `m.*` messages.
5. **Include `sitemap.xml`** plus a `Sitemap:` line in `robots.txt`.

## Architecture & rendering

- **New route** `frontend/src/routes/about/+page.svelte` — renders the About
  content (the current modal body) using `m.aboutBody({ siteName })` and
  `$config.siteName`. The `config` store already defaults to `{ siteName:
  'Go-Stop', ... }` and `loadConfig()` is a no-op when `!browser`, so prerender
  uses the default and client hydration updates the real instance name.
- **New** `frontend/src/routes/about/+page.ts`:
  ```ts
  export const prerender = true;
  export const ssr = true;
  ```
  These override the root layout's `ssr=false`/`prerender=false` for this route
  only (more-deeply-nested page options win in SvelteKit). Other routes stay SPA.
- `adapter-static` with `trailingSlash: 'never'` emits **`web/build/about.html`**
  with the French About text baked into the HTML.

## SSR-hardening

The layout tree is already prerender-aware:
- `pwa.ts` and `locale.ts` guard browser access behind `typeof localStorage !==
  'undefined'` and call browser APIs only inside `onMount`/handlers.
- `A2HSBanner.svelte` guards `localStorage` with `browser`.
- `LangPicker.svelte` only touches `localStorage` on click.

Plan: enable SSR for `/about`, run `npm run build`, and fix any prerender-time
errors surfaced by always-rendered components (expected: few or none). The one
unknown is `svelte-persisted-store` (`userPhone`/`userName`) at module init —
verify it initializes safely server-side; add a `browser` guard only if it throws.

## Go serving change

`spaHandler` (`main.go`) currently serves an exact file if it exists, else falls
back to `index.html` with server-injected OG tags. For `GET /about` the file
`about` does not exist (the build emits `about.html`), so it would wrongly fall
back to the SPA shell.

Change: when no exact file matches, try `<file>.html` and serve it directly if it
exists, before the `index.html` fallback. This serves the prerendered
`about.html` for `/about` (and any future prerendered route). Because the
prerendered page carries its own `<title>`/`<meta description>`/canonical via
`<svelte:head>`, it is served as-is with **no OG injection** (no double `<title>`).

Guardrails to preserve: keep the existing `buildDir` path-escape check; only the
`.html` retry is added.

Additionally register two explicit `GET` routes **before** the `NoRoute`
fallback, so they take precedence over static-file serving:
- `GET /sitemap.xml` → dynamic sitemap handler (see SEO section).
- `GET /robots.txt` → dynamic robots handler (see SEO section), replacing the
  static file.

## Replace the modal

- `TopBar.svelte`: the ⓘ control becomes an anchor `href="/about"` (real link, so
  it is crawlable and middle-clickable) instead of an `onabout` callback button.
- `+layout.svelte`: remove `AboutModal` import, the `showAbout` state, and the
  `<AboutModal/>` element; drop the `onabout` wiring.
- Delete `frontend/src/lib/components/layout/AboutModal.svelte`.
- Add an **About** link to the footer in `+layout.svelte` (internal link aids
  crawl + discoverability), alongside Privacy · Stats.

## SEO specifics

- `<svelte:head>` on `/about`:
  - localized `<title>` (e.g. `About · {siteName}` via a message key),
  - `<meta name="description">` (localized tagline),
  - `<link rel="canonical" href="/about">` — **relative**. The page is
    prerendered at build time when the deployment host is unknown, and this is a
    deploy-your-own app (every instance has a different domain), so a relative
    canonical (which Google accepts) is correct; an absolute one would be wrong
    or stale across instances.
- Fold in the **Scalingo deploy badge**: replace the text "▶ Deploy on Scalingo"
  link with the official badge image
  `https://cdn.scalingo.com/deploy/button.svg` inside the existing
  `deploy?source=…#main` anchor (alt text stays localized). Applies to all six
  `aboutBody` locale strings.
- **`sitemap.xml` — served dynamically by Go** at `GET /sitemap.xml`, building
  **absolute** `<loc>` URLs from the request scheme + host (reuse `og.go`'s
  `ogScheme`/`ogAbsURL` helpers). Lists `/`, `/about`, `/stats`. Dynamic
  generation is required because each deployed instance has its own host, unknown
  at build time.
- **`robots.txt` — served dynamically by Go** at `GET /robots.txt`: keep the
  allow-all body and append `Sitemap: <scheme>://<host>/sitemap.xml` (absolute,
  per the robots spec) using the same host helpers. This replaces the current
  static `frontend/static/robots.txt`, which is removed (an explicit Go route
  takes precedence over the static-file fallback anyway).

### Out of scope (YAGNI)

Per-language indexed URLs / `hreflang` / language-specific static pages. There is
one canonical `/about` (French snapshot) that localizes client-side for users.
Can be added later if multi-language SEO is wanted.

## Content / i18n

Reuse the existing `aboutBody` strings (all six locales) — no copy duplication.
New message keys only for the page `<title>`/description if not already present.

## Testing

- **Go** (`main_serve_test.go`): add cases — `GET /about` serves the `about.html`
  body (not the index fallback); a path with no file and no `.html` sibling still
  falls back to `index.html`; the existing exact-file and `/api` 404 behavior is
  unchanged. `GET /sitemap.xml` returns XML with absolute `<loc>` URLs derived
  from the request host; `GET /robots.txt` returns the allow-all body with an
  absolute `Sitemap:` line.
- **Build/prerender**: assert `web/build/about.html` exists and contains the
  About body text (French) in raw HTML.
- **Frontend**: `svelte-check` clean. e2e has no About references today, so
  nothing breaks; optionally add an e2e asserting `/about` renders and the ⓘ icon
  navigates there.
- **Manual**: Playwright view of `/about` and the ⓘ → `/about` navigation; confirm
  language switching still localizes the page.

## Risks / open items

- `svelte-persisted-store` SSR init (mitigation: `browser` guard if needed).
- Enabling SSR on one route while the app is otherwise SPA — verify the build
  succeeds and other routes remain client-rendered.
- The prerendered snapshot shows the default `siteName` ("Go-Stop"); acceptable
  since the live page corrects it on hydration and the SEO value is the generic
  product description.
