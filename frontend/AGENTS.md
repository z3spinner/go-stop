# AGENTS.md — `frontend/` (SvelteKit SPA)

Svelte 5 + SvelteKit 2 + TypeScript, Tailwind v4, shadcn-svelte, Paraglide i18n.
Builds to a **static SPA** at `../web/build`, which the Go server serves. There is
no SSR (`src/routes/+layout.ts` sets `ssr = false; prerender = false`).

## Commands (run from `frontend/`)

| Task | Command |
|---|---|
| Dev server (Vite, proxies `/api`→:8080) | `npm run dev` |
| Type-check | `npm run check` |
| Unit/component tests | `npm run test:unit` (or `npm test` for a single run) |
| Build → `../web/build` | `npm run build` |
| Regenerate API client | `npm run api:generate` (or repo-root `make api-generate`) |
| Compile i18n messages | `npm run paraglide` |

`vite.config.ts` proxies `/api` to `VITE_API_PROXY_TARGET` (default
`http://localhost:8080`). The compose devstack runs this on :5173.

## Generated API client — do not hand-edit

`src/lib/api/generated/go-stop-api.ts` is **orval-generated** from the backend
OpenAPI spec (`docs/swagger.json`). To change it, change the backend annotations
and run `make api-generate`. Consume the API only through the hand-written layers
on top of it:

- `src/lib/api/fetchMutator.ts` — custom fetch: prefixes `/api`, unwraps the
  `{ error }` envelope into a thrown `ApiError(status, message)`, maps 204→null.
- `src/lib/api.ts` — the stable `api.<resource>.<verb>()` facade used by
  components; injects the `X-Phone` header for owner-scoped reads.
- `src/lib/types.ts` — friendly aliases over generated models, wrapped in
  `DeepRequired<T>` (the server always populates fields the spec marks optional).

Import `api` from `$lib/api`, not the generated module directly.

## Components

- `src/lib/components/ui/*` — **shadcn-svelte primitives** (button, dialog, card,
  select, tabs, …). Treat as generated: add/update via the shadcn CLI
  (`components.json` holds the config), don't hand-roll them.
- `src/lib/components/{rides,requests,alerts,notifications,layout}/` — domain
  components. Co-located `*.test.ts` next to each.
- **Svelte 5 runes throughout** — `$props`, `$state`, `$derived`, `{@render ...}`,
  `$bindable`. Don't introduce Svelte 4 idioms (`export let`, `$:`, slots).

## i18n (Paraglide)

- Messages: `src/messages/{fr,en,es,it,de,nl}.json`. **`fr` is the base locale**
  (`project.inlang/settings.json`). Compiled output lands in `src/lib/paraglide/`
  (gitignored) — run `npm run paraglide` if message functions look stale.
- Use in components: `import { m } from '$lib/paraglide/messages'` then
  `m.someKey()` / `m.someKey({ count })`.
- Locale is stored **raw** (unquoted) in `localStorage["lang"]` by the custom
  strategy in `src/lib/locale.ts`.
- `src/messages/messages.test.ts` enforces that **all locales share identical
  keys with no empty values** — add every new key to all six files or the suite
  fails.

## Routing

File-based under `src/routes/` (`+page.svelte`, optional `+page.ts` loader,
`+layout.svelte` shell). Pages include `/`, `/search`, `/stats`, `/post-ride`,
`/post-request`, `/me`, `/my-rides`, `/my-searches`, and detail routes
`/rides/[id]`, `/rides/[id]/feedback`, `/requests/[id]`, `/interests/[id]`.
SPA fallback is `index.html`; prefer `goto('/')` over `history.back()`.

## State

- `src/lib/stores.ts` — persisted profile stores (`userName`, `userPhone`,
  `lastOrigin`, `lastDestination`) via `svelte-persisted-store` with a **raw
  string serializer** (legacy-compatible; values are not JSON-quoted). E2E and the
  app both rely on this raw format.
- `src/lib/pwa.ts` / `a2hs.ts` — Web Push subscription, notification polling, and
  add-to-home-screen state; `static/sw.js` is the (hand-written, un-bundled)
  service worker.

## Testing

Vitest + jsdom + `@testing-library/svelte`; setup in `vitest-setup.js`. Tests are
co-located `*.test.ts`. Globals are enabled (no need to import `describe/it/expect`).
Clear `localStorage` in `beforeEach` when a test touches stores.

## Licensing headers

New `.ts`/`.js` files start with the two-line `//` SPDX header; new `.svelte`
files start with the `<!-- -->` SPDX comment block (see root `AGENTS.md` →
"Licensing & file headers"). Do **not** add headers to generated/vendored code:
`src/lib/api/generated/`, the shadcn `src/lib/components/ui/` primitives, or
`src/lib/paraglide/`.
