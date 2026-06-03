# Frontend Refactor (SvelteKit + TS + shadcn-svelte + Paraglide) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 2489-line vanilla-JS `web/js/app.js` with a typed SvelteKit SPA (TypeScript + Vite + shadcn-svelte + Paraglide), served as a static build by the unchanged Go backend, with all 32 Playwright e2e tests passing.

**Architecture:** SvelteKit with `@sveltejs/adapter-static` in SPA mode (`ssr = false`, `fallback: index.html`) builds to `web/build/`. The Go (Gin) server serves that directory and keeps the existing `/api/*` routes untouched. In dev, Vite (port 5173) proxies `/api` → Go (port 8080). State lives in `localStorage` (keys preserved from today for user-data continuity). i18n is compile-time via Paraglide; the active locale persists in the existing `localStorage["lang"]` key.

**Tech Stack:** SvelteKit (Svelte 5 runes), TypeScript (strict), Vite, Tailwind CSS, shadcn-svelte (Bits UI), Paraglide JS (`@inlang/paraglide-js`), Vitest + @testing-library/svelte (jsdom), Playwright (existing suite).

---

## How to read this plan

- **Decisions & Deviations** (below) record where this plan intentionally differs from `docs/superpowers/specs/2026-06-02-frontend-refactor-design.md`, and why. Read these first — they change a few things the spec implies.
- **Appendix A** is the complete typed API client. **Appendix B** is the complete set of 6 Paraglide message files. **Appendix C** is the Test-Hook Contract (every DOM id/class/`name` attribute the e2e suite selects — these MUST be reproduced). **Appendix D** is the localStorage key map. Tasks reference these appendices instead of repeating large content.
- Field-name casing in API responses is **inconsistent and load-bearing** (PascalCase for Ride/Request entities, snake_case for interests/notifications, camelCase for config/vapid). Appendix A encodes the exact casing. Do not "normalize" it.

---

## Decisions & Deviations from the spec

These are deliberate. Each has a rationale grounded in the existing code or the e2e contract.

**D1 — e2e runs against the Go-served build on `:8080`, not the Vite dev server on `:5173`.**
The spec §9 says "playwright.config.ts updated to run against Vite dev server on port 5173." The existing suite hardcodes `const BASE = 'http://localhost:8080'` and issues same-origin `fetch('/api/...')` calls inside `page.evaluate()`. Running against the real built artifact served by Go (a) keeps `BASE` and same-origin `/api` working with zero proxy, (b) tests the actual production artifact, and (c) avoids a class of dev-only discrepancies. The Playwright `webServer` builds the frontend and starts Go before the run. (Dev humans still use Vite on 5173 via `make dev`.)

**D2 — localStorage keys are preserved exactly; profile is NOT consolidated into a single `profile` object.**
The spec §5 proposes `persisted('profile', {name, phone})` and `persisted('lang', 'fr')`. But the live app uses separate keys `user_name`, `user_phone`, `lang`, `last_origin`, `last_destination`, `interest_<rideID>`, `a2hs_dismissed`, `standalone_notif_prompted`, `poll_seen` — and e2e test 30 asserts `localStorage['user_name']` / `localStorage['user_phone']` directly, while tests seed identity through these keys. Changing the key shape would silently log out every existing user and break the e2e identity contract. We keep the exact keys (Appendix D). `stores.ts` therefore exposes `userName`/`userPhone` as two persisted string stores, not one object.

**D3 — The 7 e2e tests that call `window.renderX()` globals are rewritten to navigate real routes.**
Tests 7, 8, 9, 10, 29, 31, 32 call in-page globals (`renderMyRides()`, `renderNotifyRoute()`, `renderMySearches()`, `renderPostRide()`, `renderMe()`) via `page.evaluate()`. SvelteKit has no such globals. Each is replaced by real navigation: e.g. `renderMyRides()` → `page.goto(BASE + '/my-rides')`. The notify/alert form (test 8's `renderNotifyRoute('Saillans','Crest')`) gets a real route — see D4.

**D4 — A `/post-request` route renders the alert form (`AlertForm`).**
The legacy app reached the alert-creation form (`renderNotifyRoute`) transiently from search results, plus a buried `/post-request` route (`renderPostRequest`). The spec's route table omits it, but the spec's component list keeps `AlertForm.svelte ← post-request form`. We give `AlertForm` a real URL: `/post-request?origin=&destination=&departure_at=`. This (a) makes test 8 navigable, (b) preserves the legacy deep link, and (c) is also rendered inline by the search page's "Notify me" button. Both entry points reuse the same `AlertForm` component.

**D5 — Two i18n bugs in the source are fixed during migration.**
(a) `btnActivate` is missing from the English block in `app.js` (English silently falls back to a hardcoded `'Activate'`). The migrated `en.json` includes `btnActivate: "Activate"`. (b) The `driver_shared` interest status renders a hardcoded French literal `'notification envoyée ✓'` for all languages (`app.js:2024`). The migration adds a translated key `notifSentShort` in all 6 languages.

**D6 — Paraglide locale persists in `localStorage["lang"]` via a custom client strategy; switching reloads.**
To honor existing users and e2e test 11 (`localStorage.lang='en'; reload; expect English`), Paraglide must read/write the `lang` key. Paraglide's built-in `localStorage` strategy uses its own key name, so we register a **custom client strategy** that reads/writes `lang` (Task 9 gives the code and a verification step against the installed Paraglide API). `baseLocale` is `fr`. The language picker calls `setLocale(code)`, which persists and reloads — matching the e2e flow and acceptably close to the legacy in-place re-render.

**D7 — Styling is Tailwind + shadcn-svelte (new look); test-hook ids/classes/`name`s are preserved as explicit attributes.**
shadcn components style via utility classes, not the legacy semantic class names. Wherever an e2e test selects by id/class/`name`, the component MUST carry that exact attribute as a stable hook (Appendix C), even when the visual styling comes from Tailwind/shadcn. We do not port `web/css/style.css`; visual identity is not an acceptance criterion, but the Test-Hook Contract is.

**D8 — The Go index template + version cache-busting plumbing is removed.**
`main.go` static serving is replaced with a single static-dir serve of `web/build` plus an SPA fallback. `internal/boundaries/handler/index_handler.go` and the `{{.Version}}` template are deleted (Vite content-hashes assets). `internal/version` is retained only for the startup log line.

**D9 — The API client and domain types are GENERATED from OpenAPI, not hand-written (mirrors the bizniz toolchain).**
Instead of a hand-maintained `api.ts`/`types.ts` (the original Appendix A), the Go handlers carry `swaggo/swag` annotations, `make swagger` emits `docs/swagger.json`, and the frontend runs `orval` to generate a typed client + models into `frontend/src/lib/api/generated/`. This makes the TS types correct *by construction* — eliminating the inconsistent-casing risk (PascalCase rides/requests vs snake_case interests) since the schema is derived from the actual Go structs. A thin, hand-written **facade** `frontend/src/lib/api.ts` re-groups the generated operations into the `api.<resource>.<verb>()` surface the components use, and `frontend/src/lib/types.ts` re-exports the generated models under the friendly names (`Ride`, `PublicRide`, …) plus the `Flexibility = 0 | 30 | 60` union. Pattern source: `/home/zeno/bizniz/dev/bizbiz-apiserver` (swag annotations + `make swagger`) and `/home/zeno/bizniz/dev/bizniz-react-frontend` (`orval.config.ts`, `client` + custom mutator, generated file consumed via an adapter layer). Task 9 is split into **Task 9a** (backend annotations + spec) and **Task 9b** (frontend orval client + facade + tests). **Appendix A below is now the FACADE CONTRACT + expected-model reference**: the generated models must match those shapes, and the `api` facade must expose exactly that surface (downstream Tasks 13–28 depend on it and are unchanged).

---

## File Structure

New SvelteKit project under `frontend/`. Created/owned by this plan:

```
frontend/
├── src/
│   ├── app.html                         ← HTML shell w/ PWA meta (replaces web/index.html)
│   ├── app.css                          ← Tailwind entry + shadcn base layer
│   ├── app.d.ts                         ← SvelteKit ambient types
│   ├── hooks.ts                         ← (reroute no-op; locale handled client-side)
│   ├── lib/
│   │   ├── api.ts                       ← typed API client (Appendix A)
│   │   ├── types.ts                     ← API domain types (Appendix A)
│   │   ├── stores.ts                    ← userName, userPhone, lastOrigin, lastDestination (persisted)
│   │   ├── pwa.ts                        ← push subscribe, A2HS, polling, bell state
│   │   ├── utils.ts                     ← formatTime, formatDate, normalizePhone, defaultDeparture, flexMinutes
│   │   ├── locale.ts                    ← Paraglide custom 'lang' strategy + helpers
│   │   ├── paraglide/                   ← GENERATED by Paraglide compiler (gitignored)
│   │   ├── components/
│   │   │   ├── layout/{TopBar,PageBar,LangPicker,AboutModal,PrivacyModal}.svelte
│   │   │   ├── rides/{RideCard,RideForm,SeekerRow,ContactOrInterest}.svelte
│   │   │   ├── alerts/{AlertCard,AlertForm}.svelte
│   │   │   ├── requests/RequestCard.svelte
│   │   │   ├── notifications/{BellButton,NotifModal,A2HSModal,A2HSBanner,PollToast}.svelte
│   │   │   └── ui/                       ← shadcn-svelte generated components
│   ├── messages/{en,fr,es,it,de,nl}.json ← Paraglide source messages (Appendix B)
│   └── routes/
│       ├── +layout.svelte               ← SPA shell: header, banner, toast, modals
│       ├── +layout.ts                   ← export const ssr = false; prerender = false
│       ├── +page.svelte                 ← / home
│       ├── post-ride/+page.svelte
│       ├── post-request/+page.svelte    ← AlertForm route (D4)
│       ├── search/+page.svelte
│       ├── my-rides/+page.svelte
│       ├── my-searches/+page.svelte
│       ├── my-alerts/+page.svelte        ← redirects to /my-searches
│       ├── my-requests/+page.svelte      ← redirects to /my-searches
│       ├── me/+page.svelte
│       ├── stats/+page.svelte
│       ├── interests/[id]/+page.svelte
│       ├── rides/[id]/+page.svelte        ← push deep link
│       └── requests/[id]/+page.svelte     ← push deep link
├── static/                              ← copied verbatim to web/build root
│   ├── sw.js  manifest.json  logo.svg  icon-192.png  icon-512.png
│   ├── icon-maskable-192.png  icon-maskable-512.png  apple-touch-icon.png
├── project.inlang/settings.json         ← Paraglide/inlang config
├── components.json                      ← shadcn-svelte config
├── svelte.config.js  vite.config.ts  tsconfig.json  package.json
```

Repo-root files modified by this plan:

```
main.go                                  ← static serving rewrite (Task 4)
internal/boundaries/handler/index_handler.go  ← DELETED (Task 4)
.buildpacks                              ← NEW: nodejs then go (Task 3)
package.json (root)                      ← build delegates to frontend/ (Task 3)
.gitignore                               ← add web/build, frontend/.svelte-kit, frontend/src/lib/paraglide (Task 3)
Dockerfile                               ← add Node build stage (Task 3)
Makefile                                 ← dev / build-web / test-e2e targets (Task 3)
playwright.config.js                     ← webServer + run against :8080 (Task 30)
e2e/gostop.spec.js                       ← rewrite 7 global-fn tests, update selectors (Task 30)
web/js/app.js, web/css/style.css, web/index.html, web/js/sw.js  ← DELETED (Task 31)
```

---

## Appendix A — Domain types & API client (exact, copy-paste)

### `src/lib/types.ts`

> Casing mirrors the Go wire format exactly. `Ride`/`Request`/`PublicRide`/`PublicRequest` and `Stats.top_routes[]` are **PascalCase** (Go structs without json tags). Interest/notification/contact shapes are **snake_case**. `Config`/`VapidKey` are **camelCase**. Timestamps are RFC3339 strings.

```typescript
// Flexibility is a numeric enum: 0 = exact, 30 = ±30 min, 60 = ±60 min.
export type Flexibility = 0 | 30 | 60;

export interface Ride {
	ID: string;
	DriverName: string;
	Phone: string; // present on full Ride (create / get-by-id / my-rides)
	Origin: string;
	Destination: string;
	Date: string;
	DepartureAt: string;
	Flexibility: Flexibility;
	PostedAt: string;
	ExpiresAt: string;
	FeedbackGiven: boolean;
}

export interface PublicRide {
	ID: string;
	DriverName: string;
	Origin: string;
	Destination: string;
	Date: string;
	DepartureAt: string;
	Flexibility: Flexibility;
	PostedAt: string;
	ExpiresAt: string;
	FeedbackGiven: boolean;
	InterestCount: number; // no Phone in public shape
}

export interface PublicRequest {
	ID: string;
	SearcherName: string;
	Origin: string;
	Destination: string;
	DepartureAt: string;
	Flexibility: Flexibility;
}

export interface Request {
	ID: string;
	SearcherName: string;
	Phone: string;
	Origin: string;
	Destination: string;
	Date: string; // zero-value "0001-01-01T00:00:00Z" => anytime/no-date
	DepartureAt: string; // zero-value => no time; 1970-01-01 sentinel => daily time-only
	Flexibility: Flexibility;
	PostedAt: string;
	ExpiresAt: string;
}

export type InterestStatus = 'pending' | 'accepted' | 'driver_shared';

export interface InterestListItem {
	id: string;
	status: InterestStatus;
	searcher_name?: string;
	searcher_phone?: string; // only when status === 'accepted'
}

export interface MyInterest {
	id: string;
	ride_id: string;
	status: InterestStatus;
	driver_name: string;
	origin: string;
	destination: string;
	departure_at: string;
}

export interface ContactInfo {
	phone: string;
	name: string;
	role: 'driver' | 'searcher';
	origin: string;
	destination: string;
	departure_at: string;
}

export interface ExpressInterestResponse { id: string; status: InterestStatus; }
export interface AcceptInterestResponse { searcher_phone: string; }

export interface NotificationItem {
	ride_id: string;
	request_id: string;
	driver_name: string;
	origin: string;
	destination: string;
	departure_at: string;
	sent_count: number;
}

export interface RouteStat { Origin: string; Destination: string; Count: number; }
export interface ActivityCounts { all_time: number; this_year: number; this_month: number; }
export interface Stats {
	top_routes: RouteStat[];
	total_confirmed: number;
	total_this_week: number;
	searches: ActivityCounts;
	rides_posted: ActivityCounts;
}

export interface Config { siteName: string; returnDelayHours: number; }
export interface VapidKey { publicKey: string; }

// Request bodies
export interface PostRideBody {
	driver_name: string;
	phone: string;
	origin: string;
	destination: string;
	departure_at: string; // RFC3339
	flexibility?: Flexibility;
}
export interface PostRequestBody {
	searcher_name: string;
	phone: string;
	origin: string;
	destination: string;
	departure_at?: string;   // RFC3339 (specific instant)
	departure_date?: string; // YYYY-MM-DD (whole day)
	departure_time?: string; // HH:MM (daily)
	flexibility?: Flexibility;
}
export interface SubscriptionBody { phone: string; endpoint: string; p256dh: string; auth: string; }
export interface RideSearchParams {
	origin?: string;
	destination?: string;
	departure_at?: string;  // RFC3339 UTC
	search_date?: string;   // YYYY-MM-DD
	search_time?: string;   // HH:MM (local; server converts via SERVICE_TZ)
}
```

### `src/lib/api.ts`

> Single `apiFetch` helper. Throws `ApiError` on non-2xx. Reads `{error}` envelope. `X-Phone` header for owner-scoped reads; `phone` in body for mutations. 204/empty → `null`.

```typescript
import type {
	Ride, PublicRide, PublicRequest, Request, InterestListItem, MyInterest,
	ContactInfo, ExpressInterestResponse, AcceptInterestResponse, NotificationItem,
	Stats, Config, VapidKey, PostRideBody, PostRequestBody, SubscriptionBody,
	RideSearchParams, Flexibility
} from './types';

export class ApiError extends Error {
	constructor(public status: number, message: string) {
		super(message);
		this.name = 'ApiError';
	}
}

interface FetchOpts {
	method?: 'GET' | 'POST' | 'DELETE';
	body?: unknown;
	phone?: string; // sets X-Phone header
}

async function apiFetch<T>(path: string, opts: FetchOpts = {}): Promise<T> {
	const headers: Record<string, string> = {};
	if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
	if (opts.phone) headers['X-Phone'] = opts.phone;
	const res = await fetch(`/api${path}`, {
		method: opts.method ?? 'GET',
		headers,
		body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined
	});
	if (res.status === 204) return null as T;
	const text = await res.text();
	const data = text ? JSON.parse(text) : null;
	if (!res.ok) {
		const msg = (data && typeof data === 'object' && 'error' in data) ? (data as { error: string }).error : res.statusText;
		throw new ApiError(res.status, msg);
	}
	return data as T;
}

function qs(params: Record<string, string | undefined>): string {
	const u = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) if (v) u.set(k, v);
	const s = u.toString();
	return s ? `?${s}` : '';
}

export const api = {
	config: {
		get: () => apiFetch<Config>('/config')
	},
	vapid: {
		getPublicKey: () => apiFetch<VapidKey>('/vapid-public-key')
	},
	stats: {
		get: () => apiFetch<Stats>('/stats')
	},
	destinations: {
		list: () => apiFetch<string[]>('/destinations')
	},
	rides: {
		// feed (no args) | search (origin+destination [+date/time]) | my-rides (phone)
		list: (params: RideSearchParams = {}, phone?: string) =>
			apiFetch<PublicRide[] | Ride[]>(`/rides${qs(params)}`, { phone }),
		get: (id: string) => apiFetch<Ride>(`/rides/${id}`),
		post: (body: PostRideBody) => apiFetch<Ride>('/rides', { method: 'POST', body }),
		del: (id: string, phone: string) =>
			apiFetch<null>(`/rides/${id}`, { method: 'DELETE', body: { phone } }),
		listInterests: (id: string, phone: string) =>
			apiFetch<InterestListItem[]>(`/rides/${id}/interests`, { phone }),
		listMatchingRequests: (id: string, phone: string) =>
			apiFetch<PublicRequest[]>(`/rides/${id}/requests`, { phone }),
		feedback: (id: string, phone: string, taken: boolean) =>
			apiFetch<null>(`/rides/${id}/feedback`, { method: 'POST', body: { phone, taken } })
	},
	requests: {
		list: (phone: string) => apiFetch<Request[]>('/requests', { phone }),
		get: (id: string, phone: string) => apiFetch<Request>(`/requests/${id}`, { phone }),
		post: (body: PostRequestBody) => apiFetch<Request>('/requests', { method: 'POST', body }),
		del: (id: string, phone: string) =>
			apiFetch<null>(`/requests/${id}`, { method: 'DELETE', body: { phone } }),
		ping: (id: string, rideId: string, phone: string) =>
			apiFetch<null>(`/requests/${id}/ping`, { method: 'POST', body: { ride_id: rideId }, phone })
	},
	interests: {
		express: (rideId: string, phone: string, name?: string) =>
			apiFetch<ExpressInterestResponse>(`/rides/${rideId}/interest`, { method: 'POST', body: { phone, name } }),
		accept: (interestId: string, phone: string) =>
			apiFetch<AcceptInterestResponse>(`/interests/${interestId}/accept`, { method: 'POST', body: { phone } }),
		getContact: (interestId: string, phone: string) =>
			apiFetch<ContactInfo>(`/interests/${interestId}/contact`, { phone }),
		listMine: (phone: string) => apiFetch<MyInterest[]>('/interests', { phone })
	},
	subscriptions: {
		upsert: (body: SubscriptionBody) => apiFetch<null>('/subscriptions', { method: 'POST', body }),
		remove: (phone: string) => apiFetch<null>(`/subscriptions/${encodeURIComponent(phone)}`, { method: 'DELETE' })
	},
	notifications: {
		list: (phone: string) => apiFetch<NotificationItem[]>('/notifications', { phone })
	}
};
```

---

## Appendix B — Paraglide message files

All 6 files share the **same key set**. Source of truth for the values is the `STRINGS` object in `web/js/app.js` (lines 10–863), with these **transforms** applied (Paraglide message format — Inlang Message Format):

| Legacy `STRINGS` entry | Migration |
|---|---|
| `alertCard: (r) => \`${r.Origin} → ${r.Destination}\`` | `"alertCard": "{origin} → {destination}"` |
| `aboutBody: (siteName) => \`…${esc(siteName)}…\`` | one HTML string; `${esc(siteName)}` → `{siteName}` |
| `statsAllTime: (n) => \`All time: ${n} confirmed\`` | `"statsAllTime": "All time: {n} confirmed"` (localised) |
| `statsRouteCount: (n) => \`${n} ✓\`` | `"statsRouteCount": "{n} ✓"` |
| `pendingInterests: (n) => n===1 ? '1 person interested' : \`${n} people interested\`` | `"pendingInterests": "{count, plural, one {1 person interested} other {# people interested}}"` (localised) |
| `interestCount: (n) => n===1 ? '1 request' : \`${n} requests\`` | `"interestCount": "{count, plural, one {1 request} other {# requests}}"` (localised) |
| `flexLabel: {0:'Exact',30:'±30 min',60:'±60 min'}` | three keys `flexLabelExact`, `flexLabel30`, `flexLabel60` |
| `locale: "en-GB"` etc. | **dropped from messages** — locale→BCP47 map lives in `utils.ts` (Task 7) |
| `at: "at"` | kept as `"at"` |
| `privacyBody`, `aboutBody` (HTML) | kept verbatim per language (render via `{@html}`) |

**Bug fixes during migration (D5):**
- Add `"btnActivate": "Activate"` to **en.json** (missing in `app.js` en block).
- Add `"notifSentShort"` to all 6 (replaces hardcoded French `'notification envoyée ✓'` at `app.js:2024`): en `"Notification sent ✓"`, fr `"Notification envoyée ✓"`, es `"Notificación enviada ✓"`, it `"Notifica inviata ✓"`, de `"Benachrichtigung gesendet ✓"`, nl `"Melding verzonden ✓"`.

> **Process for es/it/de/nl:** they are a mechanical value-port of the existing, verified `STRINGS[<lang>]` block — copy each value from `app.js` into the corresponding key, applying the transforms above. `en.json` and `fr.json` are given in full below as the authoritative target shape and because the e2e suite asserts their exact strings (`'Je conduis'`, `'← Retour'`, `'Mes trajets'`, `'Mon profil'`, `"I'm driving"`).

### `src/messages/en.json` (complete)

```json
{
	"$schema": "https://inlang.com/schema/inlang-message-format",
	"tagline": "Local rides, direct contact",
	"btnDriver": "I'm driving",
	"btnSearcher": "I need a ride",
	"postRideTitle": "Post a ride",
	"postReqTitle": "Post a waiting request",
	"findTitle": "Find a ride",
	"labelName": "Your name",
	"labelPhone": "Phone number",
	"labelFrom": "From",
	"labelTo": "To",
	"labelDatetime": "Date & departure time",
	"labelFlex": "Flexibility",
	"flexExact": "Exact",
	"flex30": "±30 minutes",
	"flex60": "±60 minutes",
	"btnPostRide": "Post ride",
	"btnPostReq": "Post request",
	"btnSearch": "Search",
	"btnBack": "← Back",
	"noRides": "No rides found.",
	"btnWaitingReq": "Post a waiting request",
	"privacyTitle": "Privacy",
	"privacyClose": "Close",
	"notifTitle": "Get notified of matches",
	"notifBody": "Allow notifications to be alerted when a matching ride or passenger is found.",
	"notifEnable": "Enable notifications",
	"notifSkip": "No thanks",
	"notifDenied": "Notifications blocked in browser settings.",
	"btnMyRides": "My rides",
	"btnMe": "Me",
	"meTitle": "My profile",
	"meHint": "Your name and number are saved on this device only.",
	"meSaved": "Saved ✓",
	"btnSave": "Save",
	"btnMyRequests": "My requests",
	"myRequestsTitle": "My requests",
	"noMyRequests": "No contact requests yet.",
	"reqStatusPending": "Pending",
	"reqStatusAccepted": "Accepted",
	"myRidesTitle": "My rides",
	"labelPhoneCheck": "Your phone number",
	"btnShowRides": "Show my rides",
	"noMyRides": "No active rides found for this number.",
	"btnDelete": "Delete",
	"deleteOk": "Deleted.",
	"deleteErr": "Could not delete — is that the right phone number?",
	"seekersTitle": "People looking for this ride",
	"btnPingSearcher": "Notify →",
	"noSeekers": "No one waiting yet.",
	"labelSearchDate": "Date (optional)",
	"labelSearchTime": "Time (optional)",
	"colOutbound": "Outbound",
	"colReturn": "Return",
	"noRidesCol": "No rides available.",
	"tripTypeLabel": "Trip type",
	"tripOneWay": "One way",
	"tripReturn": "Return",
	"returnSection": "Return journey",
	"labelReturnTime": "Return departure time",
	"labelReturnFlex": "Return flexibility",
	"btnNotifyRoute": "🔔 Notify me of new rides on this route",
	"notifRouteTitle": "Get notified",
	"notifRouteBody": "We'll alert you when a ride matching this route is posted. Enter your details below.",
	"notifRouteSet": "✓ You'll be notified when a matching ride appears.",
	"alertModeTime": "Specific time",
	"alertModeDay": "Any time this day",
	"alertModeAnytime": "Any time, any date",
	"alertModeDaily": "Daily at a time",
	"alertAnytimeLabel": "Always",
	"btnMySearches": "My searches",
	"mySearchesTitle": "My searches",
	"btnShowSearches": "Show",
	"btnMyAlerts": "My alerts",
	"myAlertsTitle": "My alerts",
	"btnShowAlerts": "Show my alerts",
	"btnShowRequests": "Show my requests",
	"noMyAlerts": "No active alerts found for this number.",
	"btnSeeMatches": "See available rides →",
	"alertCard": "{origin} → {destination}",
	"detailRideTitle": "Ride available",
	"detailReqTitle": "Ride request",
	"labelDriver": "Driver",
	"labelSearcher": "Passenger",
	"labelDeparture": "Departure",
	"labelContact": "Contact",
	"btnActivate": "Activate",
	"a2hsHint": "Add to Home Screen",
	"a2hsTitle": "Enable notifications on iPhone",
	"a2hsBody": "Safari on iPhone doesn't support notifications in the browser. Add this page to your Home Screen to enable them.",
	"a2hsStep1": "1. Tap the Share button ⬆",
	"a2hsStep2": "2. Tap 'Add to Home Screen'",
	"a2hsStep3": "3. Open the app from your Home Screen",
	"a2hsNote": "Requires iOS 16.4 or later.",
	"pollToastView": "View",
	"notifEnabled": "Notifications enabled ✓ — you'll be alerted for new rides and accepted contacts.",
	"notifDeniedTip": "Notifications are blocked. Enable them in your browser settings and reload.",
	"footerPrivacy": "Privacy",
	"aboutTitle": "About Go Stop",
	"aboutBody": "<p><strong>Go Stop</strong> is a local ride-sharing platform, positioned between hitchhiking and carpooling. It connects drivers offering one-time trips with people looking for a lift. Direct contact by phone — no accounts required.</p>\n<h3>Your community</h3>\n<p>This instance is deployed for <strong>{siteName}</strong>.</p>\n<h3>Deploy for your community</h3>\n<p>Go Stop is free and open source. Deploy your own instance in one click:</p>\n<p><a href=\"https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop\" target=\"_blank\" rel=\"noopener\">▶ Deploy on Scalingo</a></p>\n<p style=\"font-size:0.8rem\">Source: <a href=\"https://github.com/z3spinner/go-stop\" target=\"_blank\" rel=\"noopener\">github.com/z3spinner/go-stop</a> · AGPL-3.0 licence</p>",
	"feedbackTitle": "Did anyone join your ride?",
	"feedbackYes": "Yes, someone joined",
	"feedbackNo": "No, I drove alone",
	"feedbackThanks": "Thanks!",
	"statsTitle": "This week",
	"statsEmpty": "No confirmed rides yet this week.",
	"statsAllTime": "All time: {n} confirmed",
	"btnAllStats": "All stats →",
	"homeFeedTitle": "Available now",
	"noActiveRides": "No rides posted yet.",
	"btnInterest": "Request contact",
	"interestSent": "Request sent — you'll be notified when the driver accepts.",
	"interestPending": "Waiting for driver",
	"btnResend": "Request again",
	"pendingInterests": "{count, plural, one {1 person interested} other {# people interested}}",
	"interestCount": "{count, plural, one {1 request} other {# requests}}",
	"btnAccept": "Accept & share my number",
	"contactRevealed": "Contact accepted",
	"theirNumber": "Their number:",
	"theirName": "Name:",
	"btnCallNow": "Call now",
	"btnSearchRoute": "Search this route",
	"statsPageTitle": "Stats",
	"statsSearches": "Searches",
	"statsRidesPosted": "Rides posted",
	"statsAllTime2": "All time",
	"statsThisYear": "This year",
	"statsThisMonth": "This month",
	"statsRouteCount": "{n} ✓",
	"privacyBody": "<h3>What we collect</h3>\n<p>When you post a ride or request we store: your name, phone number, origin, destination, departure time, and flexibility window. Nothing else.</p>\n<h3>How long we keep it</h3>\n<p>Rides and requests are <strong>automatically and permanently deleted</strong> at the end of their departure day. If you want to delete your post sooner, use the delete option — you'll need the phone number you posted with.</p>\n<p>Push notification subscriptions are kept until you unsubscribe.</p>\n<h3>Who can see your phone number</h3>\n<p>Your phone number is visible to anyone who views your ride or request card. This is intentional — it's how the two parties contact each other directly.</p>\n<h3>Cookies &amp; local storage</h3>\n<p>No cookies. Go-Stop uses no tracking and no analytics.</p>\n<p>The following is saved in your browser's <code>localStorage</code> (on your device only, never sent to the server):</p>\n<ul>\n<li>Your name and phone number — to pre-fill forms on your next visit</li>\n<li>Your language preference</li>\n</ul>\n<p>You can clear this at any time by clearing your browser's site data.</p>\n<h3>Third parties</h3>\n<p>No data is shared with third parties. Push notifications are delivered via the Web Push standard through your browser's push service (e.g. Google FCM for Chrome). The push payload contains only the match details you'd see on screen.</p>",
	"flexLabelExact": "Exact",
	"flexLabel30": "±30 min",
	"flexLabel60": "±60 min",
	"at": "at",
	"notifSentShort": "Notification sent ✓"
}
```

### `src/messages/fr.json` (complete)

```json
{
	"$schema": "https://inlang.com/schema/inlang-message-format",
	"tagline": "Trajets locaux, contact direct",
	"btnDriver": "Je conduis",
	"btnSearcher": "Je cherche un stop",
	"postRideTitle": "Proposer un trajet",
	"postReqTitle": "Publier une demande",
	"findTitle": "Trouver un stop",
	"labelName": "Votre prénom",
	"labelPhone": "Numéro de téléphone",
	"labelFrom": "Départ",
	"labelTo": "Destination",
	"labelDatetime": "Date et heure de départ",
	"labelFlex": "Flexibilité",
	"flexExact": "Exact",
	"flex30": "±30 minutes",
	"flex60": "±60 minutes",
	"btnPostRide": "Publier le trajet",
	"btnPostReq": "Publier la demande",
	"btnSearch": "Rechercher",
	"btnBack": "← Retour",
	"noRides": "Aucun trajet trouvé.",
	"btnWaitingReq": "Publier une demande",
	"privacyTitle": "Confidentialité",
	"privacyClose": "Fermer",
	"notifTitle": "Recevoir des alertes",
	"notifBody": "Activez les notifications pour être alerté(e) dès qu'un trajet ou passager correspondant est trouvé.",
	"notifEnable": "Activer les notifications",
	"notifSkip": "Non merci",
	"notifDenied": "Notifications bloquées dans les paramètres du navigateur.",
	"btnMyRides": "Mes trajets",
	"btnMe": "Moi",
	"meTitle": "Mon profil",
	"meHint": "Votre prénom et numéro sont enregistrés sur cet appareil uniquement.",
	"meSaved": "Enregistré ✓",
	"btnSave": "Enregistrer",
	"btnMyRequests": "Mes demandes",
	"myRequestsTitle": "Mes demandes",
	"noMyRequests": "Aucune demande de contact pour l'instant.",
	"reqStatusPending": "En attente",
	"reqStatusAccepted": "Acceptée",
	"myRidesTitle": "Mes trajets",
	"labelPhoneCheck": "Votre numéro de téléphone",
	"btnShowRides": "Voir mes trajets",
	"noMyRides": "Aucun trajet actif trouvé pour ce numéro.",
	"btnDelete": "Supprimer",
	"deleteOk": "Supprimé.",
	"deleteErr": "Impossible de supprimer — numéro incorrect ?",
	"seekersTitle": "Personnes cherchant ce trajet",
	"btnPingSearcher": "Prévenir →",
	"noSeekers": "Personne en attente.",
	"labelSearchDate": "Date (optionnel)",
	"labelSearchTime": "Heure (optionnel)",
	"colOutbound": "Aller",
	"colReturn": "Retour",
	"noRidesCol": "Aucun trajet disponible.",
	"tripTypeLabel": "Type de trajet",
	"tripOneWay": "Aller simple",
	"tripReturn": "Aller-retour",
	"returnSection": "Trajet retour",
	"labelReturnTime": "Heure de départ retour",
	"labelReturnFlex": "Flexibilité retour",
	"btnNotifyRoute": "🔔 Me prévenir des nouveaux trajets sur ce parcours",
	"notifRouteTitle": "Recevoir des alertes",
	"notifRouteBody": "Vous serez alerté(e) dès qu'un trajet correspondant à ce parcours est publié. Indiquez vos coordonnées.",
	"notifRouteSet": "✓ Vous serez alerté(e) dès qu'un trajet correspondant apparaît.",
	"alertModeTime": "Heure précise",
	"alertModeDay": "Toute la journée",
	"alertModeAnytime": "Toujours",
	"alertModeDaily": "Chaque jour",
	"alertAnytimeLabel": "Toujours",
	"btnMySearches": "Mes recherches",
	"mySearchesTitle": "Mes recherches",
	"btnShowSearches": "Voir",
	"btnMyAlerts": "Mes alertes",
	"myAlertsTitle": "Mes alertes",
	"btnShowAlerts": "Voir mes alertes",
	"btnShowRequests": "Voir mes demandes",
	"noMyAlerts": "Aucune alerte active trouvée pour ce numéro.",
	"btnSeeMatches": "Voir les trajets disponibles →",
	"alertCard": "{origin} → {destination}",
	"detailRideTitle": "Trajet disponible",
	"detailReqTitle": "Demande de trajet",
	"labelDriver": "Conducteur",
	"labelSearcher": "Passager",
	"labelDeparture": "Départ",
	"labelContact": "Contact",
	"btnActivate": "Activer",
	"a2hsHint": "Ajouter à l'écran d'accueil",
	"a2hsTitle": "Activer les notifications",
	"a2hsBody": "Pour recevoir des notifications sur iPhone, ajoutez cette page à votre écran d'accueil.",
	"a2hsStep1": "1. Appuyez sur le bouton Partager",
	"a2hsStep2": "2. Appuyez sur 'Sur l'écran d'accueil'",
	"a2hsStep3": "3. Ouvrez l'app depuis votre écran d'accueil",
	"a2hsNote": "Nécessite iOS 16.4 ou version ultérieure.",
	"pollToastView": "Voir",
	"notifEnabled": "Notifications activées ✓ — vous serez alerté(e) pour les nouveaux trajets et les contacts acceptés.",
	"notifDeniedTip": "Notifications bloquées. Activez-les dans les paramètres de votre navigateur puis rechargez.",
	"footerPrivacy": "Confidentialité",
	"aboutTitle": "À propos de Go Stop",
	"aboutBody": "<p><strong>Go Stop</strong> est une plateforme locale de covoiturage, à mi-chemin entre l'autostop et le covoiturage formel. Elle met en relation des conducteurs qui proposent un trajet ponctuel et des personnes qui cherchent un stop.</p>\n<p>Aucun compte n'est requis. Le contact se fait directement par téléphone.</p>\n<h3>Votre communauté</h3>\n<p>Cette instance est déployée pour <strong>{siteName}</strong>.</p>\n<h3>Déployer pour votre communauté</h3>\n<p>Go Stop est un logiciel libre. Vous pouvez déployer votre propre instance en un clic :</p>\n<p><a href=\"https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop\" target=\"_blank\" rel=\"noopener\">▶ Déployer sur Scalingo</a></p>\n<p style=\"font-size:0.8rem\">Code source : <a href=\"https://github.com/z3spinner/go-stop\" target=\"_blank\" rel=\"noopener\">github.com/z3spinner/go-stop</a> · Licence AGPL-3.0</p>",
	"feedbackTitle": "Quelqu'un est-il venu ?",
	"feedbackYes": "Oui, quelqu'un est venu",
	"feedbackNo": "Non, j'ai conduit seul(e)",
	"feedbackThanks": "Merci !",
	"statsTitle": "Cette semaine",
	"statsEmpty": "Aucun trajet confirmé cette semaine.",
	"statsAllTime": "Depuis le début : {n} confirmés",
	"btnAllStats": "Toutes les stats →",
	"homeFeedTitle": "Disponibles maintenant",
	"noActiveRides": "Aucun trajet publié pour l'instant.",
	"btnInterest": "Demander le contact",
	"interestSent": "Demande envoyée — vous serez alerté(e) lorsque le conducteur accepte.",
	"interestPending": "En attente du conducteur",
	"btnResend": "Redemander",
	"pendingInterests": "{count, plural, one {1 personne intéressée} other {# personnes intéressées}}",
	"interestCount": "{count, plural, one {1 demande} other {# demandes}}",
	"btnAccept": "Accepter et partager mon numéro",
	"contactRevealed": "Contact accepté",
	"theirNumber": "Leur numéro :",
	"theirName": "Prénom :",
	"btnCallNow": "Appeler maintenant",
	"btnSearchRoute": "Rechercher ce trajet",
	"statsPageTitle": "Statistiques",
	"statsSearches": "Recherches",
	"statsRidesPosted": "Trajets publiés",
	"statsAllTime2": "Depuis le début",
	"statsThisYear": "Cette année",
	"statsThisMonth": "Ce mois-ci",
	"statsRouteCount": "{n} ✓",
	"privacyBody": "<h3>Ce que nous collectons</h3>\n<p>Lorsque vous publiez un trajet ou une demande, nous enregistrons : votre prénom, numéro de téléphone, lieu de départ, destination, heure de départ et flexibilité. Rien d'autre.</p>\n<h3>Durée de conservation</h3>\n<p>Les trajets et demandes sont <strong>supprimés automatiquement et définitivement</strong> à la fin du jour de départ. Pour supprimer votre annonce plus tôt, utilisez l'option de suppression — le numéro de téléphone utilisé lors de la publication sera demandé.</p>\n<p>Les abonnements aux notifications push sont conservés jusqu'à ce que vous vous désinscriviez.</p>\n<h3>Qui peut voir votre numéro de téléphone</h3>\n<p>Votre numéro est visible par toute personne qui consulte votre annonce. C'est volontaire — c'est ainsi que les deux parties se contactent directement.</p>\n<h3>Cookies &amp; stockage local</h3>\n<p>Aucun cookie. Go-Stop n'utilise ni traceurs ni analytiques.</p>\n<p>Les informations suivantes sont enregistrées dans le <code>localStorage</code> de votre navigateur (sur votre appareil uniquement, jamais envoyées au serveur) :</p>\n<ul>\n<li>Votre prénom et numéro de téléphone — pour pré-remplir les formulaires à votre prochaine visite</li>\n<li>Votre préférence de langue</li>\n</ul>\n<p>Vous pouvez effacer ces données à tout moment en vidant les données de site de votre navigateur.</p>\n<h3>Tiers</h3>\n<p>Aucune donnée n'est partagée avec des tiers. Les notifications push transitent par le standard Web Push via le service push de votre navigateur (ex. Google FCM pour Chrome). Le contenu transmis se limite aux informations de mise en relation visibles à l'écran.</p>",
	"flexLabelExact": "Exact",
	"flexLabel30": "±30 min",
	"flexLabel60": "±60 min",
	"at": "à",
	"notifSentShort": "Notification envoyée ✓"
}
```

### `src/messages/{es,it,de,nl}.json`

Same key set as above. Port each value from the corresponding `STRINGS[<lang>]` block in `web/js/app.js` (lines 10–863), applying the transform table. Specifics that differ from a literal copy:

- `statsAllTime`: es `"Total: {n} confirmados"`, it `"Totale: {n} confermati"`, de `"Gesamt: {n} bestätigt"`, nl `"Totaal: {n} bevestigd"`.
- `statsRouteCount`: `"{n} ✓"` (all).
- `alertCard`: `"{origin} → {destination}"` (all).
- `pendingInterests`: es `"{count, plural, one {1 persona interesada} other {# personas interesadas}}"`, it `"{count, plural, one {1 persona interessata} other {# persone interessate}}"`, de `"{count, plural, one {1 interessierte Person} other {# interessierte Personen}}"`, nl `"{count, plural, one {1 geïnteresseerde} other {# geïnteresseerden}}"`.
- `interestCount`: es `"{count, plural, one {1 solicitud} other {# solicitudes}}"`, it `"{count, plural, one {1 richiesta} other {# richieste}}"`, de `"{count, plural, one {1 Anfrage} other {# Anfragen}}"`, nl `"{count, plural, one {1 verzoek} other {# verzoeken}}"`.
- `flexLabelExact/30/60`: es `Exacto`/`±30 min`/`±60 min`; it `Esatto`/`±30 min`/`±60 min`; de `Genau`/`±30 Min`/`±60 Min`; nl `Exact`/`±30 min`/`±60 min`.
- `at`: es `"a las"`, it `"alle"`, de `"um"`, nl `"om"`.
- `notifSentShort`: es `"Notificación enviada ✓"`, it `"Notifica inviata ✓"`, de `"Benachrichtigung gesendet ✓"`, nl `"Melding verzonden ✓"`.
- `aboutBody`/`privacyBody`: copy the es/it/de/nl HTML strings verbatim from `app.js`; in `aboutBody` replace `${esc(siteName)}` with `{siteName}`.

A Vitest test (Task 6) asserts all 6 files have an identical key set and zero missing keys, so a typo or omission fails the build.

---

## Appendix C — Test-Hook Contract

Every entry below is selected by the e2e suite. The owning component MUST render the exact id/class/attribute, regardless of styling. Treat this as a contract: each component task lists the hooks it owns; this appendix is the master list. (Where the legacy class is also a styling class, keep it as a plain class on the element; shadcn/Tailwind utilities can coexist.)

**Element IDs** (render exactly):
| id | Owner | Used by tests |
|---|---|---|
| `#app` | `+layout.svelte` wraps page content in `<div id="app">` | 7 (innerText scan for searcher name) |
| `#back` | `PageBar` back button | 32 |
| `#btn-me` | `TopBar` "me" icon link | 30 |
| `#btn-return` | `RideForm` return-toggle button | 10 |
| `#search-form` | `/search` form | 26 |
| `#notify-form` | `AlertForm` form | 8, 19, 20 |
| `#my-rides-form` | `/my-rides` gate form | 29 |
| `#my-searches-form` | `/my-searches` gate form | 9 |
| `#my-alerts-list` | `/my-searches` alerts list container | 9 (asserts `children.length > 0`) |
| `#me-form` | `/me` form | 31, 32 |
| `#me-saved` | `/me` saved indicator; toggles inline `style="display:none"` ↔ visible | 30, 32 (`#me-saved:not([style*="none"])`) |

**Classes** (render exactly):
| class | Owner | Notes |
|---|---|---|
| `.btn-primary` | primary CTA buttons | `button.btn-primary` = "Je conduis" on home (tests 1,3,11,32); also post-ride submit |
| `.btn-secondary` | secondary CTA | home "Je cherche un stop" (test 1) |
| `.btn-ghost-inline` | home ghost nav tiles (me / my-rides / my-searches) | test 1 |
| `.card` | every ride/alert/request/seeker list card | tests 5,6,7,9,17 … |
| `.card-route` | route line inside a ride card | test 3 asserts text `Saillans → Crest` |
| `.btn-interest` | request-contact button; also carries `data-ride-id` | tests 5,6 |
| `.interest-state` | interest confirmation text element | test 6 |
| `.btn-mode` | alert-mode buttons; carry `data-mode="time\|day\|daily\|anytime"` | test 8 |
| `.btn-delete` | delete button on alert/ride card | test 9 |
| `.delete-msg` | delete result message element | test 9 |
| `.results-col-header` | search results column header (2 of them) | tests 15,16,17,18 |
| `.col-notify` | "notify me" button inside a results column | tests 16–20 |
| `.col-empty` | empty-column placeholder | test 18 |
| `.btn-ping-searcher` | "Prévenir" button in seeker row | test 29 |
| `.seeker-row` | matching-searcher row in my-rides | test 29 |

**Input `name` attributes** (render exactly on the matching control):
- RideForm: `driver_name`, `phone`, `origin`, `destination`, `departure_at`, `return_departure_at`.
- AlertForm: `searcher_name`, `phone`, `origin`, `destination`, `alert_date`, `alert_time`.
- SearchForm: `origin`, `destination`, `search_date`, `search_time`.
- Me form: `name`, `phone`.
- Every form's submit control is a `button[type=submit]`.

**`data-*` attributes:** `data-ride-id` on `.btn-interest`; `data-mode` on `.btn-mode`. (Robust — keep.)

**Robust hooks the suite also relies on (preserve behavior, not markup):**
- Page `<title>` = value of `/api/config` `siteName` + the app name as today → renders `'Go Stop Saillans!'` in the test DB. Set `document.title` from config (Task 11).
- Visible texts: `'Statistiques'` (footer link, test 2), `'← Retour'` (back button, test 2), `'Je conduis'`/`"I'm driving"` (test 11), driver names present / phone numbers absent on public cards (tests 5,6,29).
- Routes/URLs preserved: `/post-ride`, `/search?origin=&destination=&departure_at=&search_date=&search_time=` (querystring round-trips on reload), `/me`. Use SvelteKit `$page.url.searchParams` + `goto` with the same param names.
- `<h1>` contains `siteName` (tests 1,2); `<h2>` per page (tests 2,3 `/My rides|Mes trajets/i`, 30 `/Mon profil|My profile|profil/i`).

---

## Appendix D — localStorage key map (preserve exactly)

| Key | Type | Owner | Notes |
|---|---|---|---|
| `user_name` | string | `stores.userName` | profile name; test 30 asserts |
| `user_phone` | string | `stores.userPhone` | profile phone; test 30 asserts |
| `lang` | string (fr/en/es/it/de/nl) | Paraglide custom strategy (`locale.ts`) | test 11 sets it + reloads |
| `last_origin` | string | `stores.lastOrigin` | search pre-fill |
| `last_destination` | string | `stores.lastDestination` | search pre-fill |
| `interest_<rideID>` | string (interest id) | search/home interest flow (direct `localStorage`) | tests 6 read/clear |
| `a2hs_dismissed` | `'1'` | A2HSBanner | iOS banner dismissed |
| `standalone_notif_prompted` | `'1'` | pwa.ts | one-time standalone prompt |
| `poll_seen` | JSON `string[]` (cap 100) | pwa.ts | notification poll dedupe |

`interest_<rideID>` is per-ride and dynamic; it is read/written with raw `localStorage` (not a persisted store). All `localStorage` access must be `browser`-guarded (`import { browser } from '$app/environment'`).

---

# Phase 0 — Scaffold & build pipeline

Outcome of Phase 0: an empty SvelteKit SPA that the Go binary builds and serves at `http://localhost:8080`, with `make dev` running both servers. No app features yet.

### Task 1: Scaffold the SvelteKit project

**Files:**
- Create: `frontend/` (entire SvelteKit project via CLI)
- Create: `frontend/src/lib/smoke.test.ts`

- [ ] **Step 1: Create the project**

Run from repo root:
```bash
npx sv create frontend --template minimal --types ts
cd frontend
npx sv add tailwindcss --no-install
npm install
npx sv add vitest --no-install
npm install
```
(If `sv` prompts interactively, choose: minimal template, TypeScript syntax, Tailwind CSS, Vitest. Decline Playwright/ESLint/Prettier here — Playwright already exists at repo root.)

- [ ] **Step 2: Write a smoke test**

`frontend/src/lib/smoke.test.ts`:
```typescript
import { describe, it, expect } from 'vitest';

describe('smoke', () => {
	it('runs vitest', () => {
		expect(1 + 1).toBe(2);
	});
});
```

- [ ] **Step 3: Run the smoke test**

Run: `cd frontend && npm run test:unit -- --run` (the `sv add vitest` add-on creates a `test:unit` script; if the script is named `test`, use that).
Expected: 1 passing test.

- [ ] **Step 4: Verify the dev server boots**

Run: `cd frontend && npm run dev` — confirm it serves on `http://localhost:5173`, then stop it (Ctrl-C).
Expected: "VITE ready" with a local URL, no errors.

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): scaffold SvelteKit + TS + Tailwind + Vitest"
```

---

### Task 2: Configure adapter-static (SPA) and the HTML shell

**Files:**
- Modify: `frontend/package.json` (swap adapter dependency)
- Modify: `frontend/svelte.config.js`
- Create: `frontend/src/routes/+layout.ts`
- Modify: `frontend/src/app.html`
- Create: `frontend/src/routes/+page.svelte` (temporary placeholder)

- [ ] **Step 1: Install the static adapter**

Run: `cd frontend && npm i -D @sveltejs/adapter-static && npm uninstall @sveltejs/adapter-auto`

- [ ] **Step 2: Configure SvelteKit to build into `web/build` as an SPA**

`frontend/svelte.config.js`:
```javascript
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: '../web/build',
			assets: '../web/build',
			fallback: 'index.html',
			precompress: false,
			strict: false
		})
	}
};

export default config;
```

- [ ] **Step 3: Disable SSR/prerender (pure client SPA)**

`frontend/src/routes/+layout.ts`:
```typescript
export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';
```

- [ ] **Step 4: Write the HTML shell with PWA meta**

`frontend/src/app.html` (preserves all PWA/iOS meta from the old `web/index.html`):
```html
<!doctype html>
<html lang="fr" translate="no">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
		<link rel="manifest" href="/manifest.json" />
		<link rel="apple-touch-icon" href="/apple-touch-icon.png" />
		<meta name="theme-color" content="#2563eb" />
		<meta name="apple-mobile-web-app-capable" content="yes" />
		<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
		<meta name="google" content="notranslate" />
		%sveltekit.head%
	</head>
	<body data-sveltekit-preload-data="hover">
		<div style="display: contents">%sveltekit.body%</div>
	</body>
</html>
```

- [ ] **Step 5: Add a temporary home placeholder**

`frontend/src/routes/+page.svelte`:
```svelte
<h1>Go-Stop</h1>
```

- [ ] **Step 6: Build and verify output lands in `web/build`**

Run: `cd frontend && npm run build`
Expected: completes with no errors; `ls ../web/build` shows `index.html` and an `_app/` directory.

- [ ] **Step 7: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): adapter-static SPA build into web/build"
```

---

### Task 3: Wire dev proxy, build delegation, buildpacks, gitignore, Docker, Makefile

**Files:**
- Modify: `frontend/vite.config.ts`
- Create: `package.json` (root — overwrite the existing Playwright-only one; merge its scripts/devDeps in)
- Create: `.buildpacks`
- Modify: `.gitignore`
- Modify: `Dockerfile`
- Modify: `Makefile`

- [ ] **Step 1: Vite dev proxy `/api` → Go on 8080**

`frontend/vite.config.ts` (full replacement — the scaffold uses a Vitest `projects`/workspace structure; replace the whole `test` section with this flat one, and **keep the `@tailwindcss/vite` plugin** from the scaffold or Tailwind v4 styling breaks):
```typescript
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': 'http://localhost:8080'
		}
	},
	test: {
		environment: 'jsdom',
		globals: true,
		setupFiles: ['./vitest-setup.js']
	}
});
```
(The `test.setupFiles` entry is created in Task 7. Also delete the scaffold's `src/lib/vitest-examples/` directory — it is unused boilerplate.)

- [ ] **Step 2: Root `package.json` delegates the production build to `frontend/`**

Overwrite `/home/zeno/dev/go-stop/package.json` (keeps the existing e2e scripts + Playwright dep, adds the build delegation the Scalingo Node buildpack runs):
```json
{
	"name": "go-stop",
	"private": true,
	"scripts": {
		"build": "npm ci --prefix frontend && npm run build --prefix frontend",
		"test:e2e": "playwright test",
		"test:e2e:ui": "playwright test --ui"
	},
	"devDependencies": {
		"@playwright/test": "^1.60.0"
	}
}
```

- [ ] **Step 3: Multi-buildpack for Scalingo (Node then Go)**

Create `/home/zeno/dev/go-stop/.buildpacks`:
```
https://github.com/Scalingo/nodejs-buildpack
https://github.com/Scalingo/go-buildpack
```

- [ ] **Step 4: Gitignore build artifacts**

Append to `/home/zeno/dev/go-stop/.gitignore`:
```
web/build/
frontend/.svelte-kit/
frontend/build/
frontend/src/lib/paraglide/
```

- [ ] **Step 5: Add a Node build stage to the Dockerfile**

Replace the production `builder`/`production` stages in `/home/zeno/dev/go-stop/Dockerfile` so the frontend is built and copied. Full updated file:
```dockerfile
# ── frontend build ──
FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN cd frontend && npm ci
COPY frontend ./frontend
RUN cd frontend && npm run build   # outputs to /app/web/build

# ── go build ──
FROM golang:1.25-alpine AS builder
ARG GIT_SHA=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN printf 'package version\n\nvar Build = "%s"\n' "${GIT_SHA}" > internal/version/build.go
RUN CGO_ENABLED=0 go build -o go-stop . && CGO_ENABLED=0 go build -o migratedb ./cmd/migratedb

# ── production ──
FROM alpine:latest AS production
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/go-stop .
COPY --from=builder /app/migratedb .
COPY --from=frontend /app/web/build ./web/build
EXPOSE 8080
CMD ["./go-stop"]

# ── development (Go hot-reload; frontend runs via vite separately) ──
FROM golang:1.25-alpine AS development
RUN go install github.com/cespare/reflex@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
EXPOSE 8080
CMD ["reflex", "-r", "\\.go$", "-s", "--", "sh", "-c", "go build -o /tmp/go-stop . && /tmp/go-stop"]
```

- [ ] **Step 6: Makefile dev/build/e2e targets**

Append to `/home/zeno/dev/go-stop/Makefile`:
```makefile
build-web:
	npm ci --prefix frontend && npm run build --prefix frontend

dev:
	@echo "Go :8080 + Vite :5173 (proxying /api). Ctrl-C stops both."
	@( go run . & echo $$! > /tmp/gostop-go.pid ; npm run dev --prefix frontend ; kill `cat /tmp/gostop-go.pid` )

test-e2e: build-web
	npm run test:e2e
```
(Tabs, not spaces, for recipe lines.)

- [ ] **Step 7: Verify build delegation works end to end**

Run: `cd /home/zeno/dev/go-stop && rm -rf web/build && npm run build`
Expected: `web/build/index.html` exists.

- [ ] **Step 8: Commit**

```bash
git add package.json .buildpacks .gitignore Dockerfile Makefile frontend/vite.config.ts
git commit -m "build: delegate frontend build, multi-buildpack, dev proxy, docker node stage"
```

---

### Task 4: Serve `web/build` from Go and remove the index template

**Files:**
- Modify: `main.go:123-139` (static serving + index handler + NoRoute)
- Delete: `internal/boundaries/handler/index_handler.go`
- Test: `main_serve_test.go` (new, repo root) — or manual verification per Step 5

- [ ] **Step 1: Replace static serving + index handler in `main.go`**

In `main.go`, delete lines 123–139 (the block from `r.Static("/css", ...)` through `r.NoRoute(indexH.Serve)`, **and** the `buildVersion := version.Get()` / `indexH, err := handler.NewIndexHandler(...)` lines). Replace with:
```go
	// Serve the SvelteKit static build. Any path that is not /api and not an
	// existing file falls back to index.html (client-side routing).
	const buildDir = "./web/build"
	log.Printf("build version: %s", version.Get())
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		clean := filepath.Clean(p)
		file := filepath.Join(buildDir, clean)
		// Guard against path traversal, then serve the file if it exists.
		if strings.HasPrefix(file, filepath.Clean(buildDir)+string(os.PathSeparator)) {
			if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
				c.File(file)
				return
			}
		}
		c.File(filepath.Join(buildDir, "index.html"))
	})
```
Ensure imports include `net/http`, `os`, `path/filepath`, `strings` (add any missing). Remove the now-unused `handler.NewIndexHandler` import usage if `handler` is otherwise still imported for API handlers (it is — keep the import). Keep `version` imported (used in the log line).

- [ ] **Step 2: Delete the index handler**

Run: `git rm internal/boundaries/handler/index_handler.go`

- [ ] **Step 3: Build the Go binary**

Run: `cd /home/zeno/dev/go-stop && go build -o /tmp/go-stop-check . && echo OK`
Expected: `OK` (compiles; no unused-import errors).

- [ ] **Step 4: Write a serving test**

`/home/zeno/dev/go-stop/main_serve_test.go`:
```go
package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildSPARouter mirrors the NoRoute SPA fallback wiring in main.go for testing.
func buildSPARouter(buildDir string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		clean := filepath.Clean(p)
		file := filepath.Join(buildDir, clean)
		if strings.HasPrefix(file, filepath.Clean(buildDir)+string(os.PathSeparator)) {
			if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
				c.File(file)
				return
			}
		}
		c.File(filepath.Join(buildDir, "index.html"))
	})
	return r
}

func TestSPAFallbackServesIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>INDEX"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := buildSPARouter(dir)

	// Deep link with no matching file → index.html
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/my-rides", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "INDEX") {
		t.Fatalf("deep link: got %d %q", w.Code, w.Body.String())
	}

	// Unknown API route → JSON 404, never index.html
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/nope", nil))
	if w.Code != 404 || strings.Contains(w.Body.String(), "INDEX") {
		t.Fatalf("api 404: got %d %q", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 5: Run the test**

Run: `go test ./... -run TestSPAFallback -count=1`
Expected: PASS.

- [ ] **Step 6: Manual end-to-end check**

Run `npm run build` (root) then `go run .`, open `http://localhost:8080` → the placeholder `Go-Stop` `<h1>` renders; `http://localhost:8080/my-rides` also serves the SPA shell; `curl -s localhost:8080/api/config` returns JSON. Stop the server.

- [ ] **Step 7: Commit**

```bash
git add main.go main_serve_test.go
git rm internal/boundaries/handler/index_handler.go
git commit -m "feat(server): serve SvelteKit build with SPA fallback; remove index template"
```

---

### Task 5: Initialise shadcn-svelte and add base UI components

**Files:**
- Create: `frontend/components.json`, `frontend/src/lib/components/ui/*`, `frontend/src/lib/utils.ts` (cn helper)

- [ ] **Step 1: Init shadcn-svelte**

Run: `cd frontend && npx shadcn-svelte@latest init --base-color slate --css ./src/app.css`
(Accept defaults for aliases: components `$lib/components`, ui `$lib/components/ui`, utils `$lib/utils`. This adds the `cn` helper and Tailwind CSS variables to `src/app.css`.)

- [ ] **Step 2: Add the components the spec lists**

Run: `cd frontend && npx shadcn-svelte@latest add -y button card input select badge dialog label`
Expected: files appear under `src/lib/components/ui/`.

- [ ] **Step 3: Verify the app still builds**

Run: `cd frontend && npm run build`
Expected: completes with no errors.

- [ ] **Step 4: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): init shadcn-svelte + base UI components"
```

---

# Phase 1 — Foundation libraries (TDD)

Outcome of Phase 1: typed, unit-tested `locale`, `utils`, `stores`, `api`, and `pwa` modules with no UI yet. Every task is test-first.

> **Vitest note:** Tasks here add a shared setup file `frontend/vitest-setup.js` (created in Task 7) referenced by `vite.config.ts`. The Paraglide compiler must run before tests that import `$lib/paraglide`; Task 6 adds an npm script `paraglide` and a `pretest` hook so generated files always exist.

### Task 6: Paraglide i18n + message files + `lang` persistence

**Files:**
- Create: `frontend/project.inlang/settings.json`
- Create: `frontend/src/messages/{en,fr,es,it,de,nl}.json` (Appendix B)
- Modify: `frontend/vite.config.ts` (add paraglide plugin)
- Create: `frontend/src/lib/locale.ts`
- Modify: `frontend/package.json` (scripts)
- Test: `frontend/src/messages/messages.test.ts`, `frontend/src/lib/locale.test.ts`

- [ ] **Step 1: Install Paraglide and create the inlang project**

Run: `cd frontend && npm i -D @inlang/paraglide-js`

`frontend/project.inlang/settings.json`:
```json
{
	"$schema": "https://inlang.com/schema/project-settings",
	"baseLocale": "fr",
	"locales": ["fr", "en", "es", "it", "de", "nl"],
	"modules": [
		"https://cdn.jsdelivr.net/npm/@inlang/plugin-message-format@latest/dist/index.js"
	],
	"plugin.inlang.messageFormat": {
		"pathPattern": "./src/messages/{locale}.json"
	}
}
```

- [ ] **Step 2: Add all 6 message files**

Create `frontend/src/messages/en.json` and `frontend/src/messages/fr.json` from Appendix B (verbatim). Create `es.json`, `it.json`, `de.json`, `nl.json` per the Appendix B porting instructions.

- [ ] **Step 3: Add the Paraglide Vite plugin (client strategies only)**

Update `frontend/vite.config.ts` to add the Paraglide plugin (keep `tailwindcss()` first and `sveltekit()` last):
```typescript
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			strategy: ['custom-lang', 'preferredLanguage', 'baseLocale']
		}),
		sveltekit()
	],
	server: { proxy: { '/api': 'http://localhost:8080' } },
	test: { environment: 'jsdom', globals: true, setupFiles: ['./vitest-setup.js'] }
});
```

- [ ] **Step 4: Add compile + pretest scripts**

In `frontend/package.json` `scripts`, add:
```json
"paraglide": "paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide",
"pretest:unit": "npm run paraglide"
```
(If the vitest script is `test` not `test:unit`, name the hook `pretest`.)

- [ ] **Step 5: Compile messages once**

Run: `cd frontend && npm run paraglide`
Expected: `src/lib/paraglide/messages.js` and `runtime.js` are generated.

- [ ] **Step 6: Write the message-parity test (fails first if a key is missing)**

`frontend/src/messages/messages.test.ts`:
```typescript
import { describe, it, expect } from 'vitest';
import en from './en.json';
import fr from './fr.json';
import es from './es.json';
import it from './it.json';
import de from './de.json';
import nl from './nl.json';

const files = { en, fr, es, it, de, nl };
const keys = (o: Record<string, unknown>) => Object.keys(o).filter((k) => k !== '$schema').sort();

describe('message files', () => {
	it('all locales share the identical key set', () => {
		const base = keys(en);
		for (const [loc, obj] of Object.entries(files)) {
			expect(keys(obj as Record<string, unknown>), `locale ${loc}`).toEqual(base);
		}
	});
	it('has no empty values', () => {
		for (const [loc, obj] of Object.entries(files)) {
			for (const [k, v] of Object.entries(obj as Record<string, string>)) {
				if (k === '$schema') continue;
				expect(String(v).length, `${loc}.${k}`).toBeGreaterThan(0);
			}
		}
	});
});
```

- [ ] **Step 7: Implement the `lang` custom locale strategy**

`frontend/src/lib/locale.ts`:
```typescript
import { browser } from '$app/environment';
import {
	defineCustomClientStrategy,
	getLocale,
	setLocale,
	locales,
	baseLocale,
	type Locale
} from '$lib/paraglide/runtime';

const KEY = 'lang';

function read(): Locale | undefined {
	if (!browser) return undefined;
	const v = localStorage.getItem(KEY);
	return v && (locales as readonly string[]).includes(v) ? (v as Locale) : undefined;
}

// Registers a Paraglide strategy named "custom-lang" that reads/writes localStorage["lang"].
// Must be called once on the client before the first getLocale() (see +layout.svelte).
export function registerLangStrategy(): void {
	if (!browser) return;
	defineCustomClientStrategy('custom-lang', {
		getLocale: () => read(),
		setLocale: (locale) => localStorage.setItem(KEY, locale)
	});
}

export { getLocale, setLocale, locales, baseLocale };
export type { Locale };
```
> **Verification fallback (Step 9):** if the installed Paraglide does not export `defineCustomClientStrategy`, implement the same contract with `overwriteGetLocale`/`overwriteSetLocale` from `$lib/paraglide/runtime`: on the client, `overwriteGetLocale(() => read() ?? baseLocale)` and `overwriteSetLocale((l) => { localStorage.setItem(KEY, l); location.reload(); })`. Either way the observable behavior (Step 8 test) is identical.

- [ ] **Step 8: Write the persistence test**

`frontend/src/lib/locale.test.ts`:
```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { registerLangStrategy } from './locale';
import { getLocale } from '$lib/paraglide/runtime';

describe('lang persistence', () => {
	beforeEach(() => localStorage.clear());

	it('reads the active locale from localStorage["lang"]', () => {
		localStorage.setItem('lang', 'en');
		registerLangStrategy();
		expect(getLocale()).toBe('en');
	});

	it('falls back to baseLocale (fr) when lang is unset/invalid', () => {
		localStorage.setItem('lang', 'zz');
		registerLangStrategy();
		expect(getLocale()).toBe('fr');
	});
});
```

- [ ] **Step 9: Run the tests; if `getLocale()` ignores the strategy, apply the Step-7 fallback, then re-run**

Run: `cd frontend && npm run test:unit -- --run src/messages src/lib/locale.test.ts`
Expected: all pass. (If the persistence test fails because the custom strategy API differs, switch `locale.ts` to the `overwriteGetLocale/overwriteSetLocale` fallback and re-run until green.)

- [ ] **Step 10: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): Paraglide i18n, 6 message files, lang persistence"
```

---

### Task 7: `utils.ts` — formatting, phone, flexibility, locale map

**Files:**
- Modify: `frontend/src/lib/utils.ts` (append to the shadcn-created file holding `cn`)
- Create: `frontend/vitest-setup.js`
- Test: `frontend/src/lib/utils.test.ts`

- [ ] **Step 1: Create the vitest setup file (jsdom + jest-dom matchers)**

Run: `cd frontend && npm i -D @testing-library/svelte @testing-library/jest-dom jsdom`

`frontend/vitest-setup.js`:
```javascript
import '@testing-library/jest-dom/vitest';
```

- [ ] **Step 2: Write the failing utils test**

`frontend/src/lib/utils.test.ts`:
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest';
import { normalizePhone, defaultDeparture, localeToBCP47, flexLabel } from './utils';

afterEach(() => vi.useRealTimers());

describe('normalizePhone', () => {
	it('strips spaces, dashes, dots and parens', () => {
		expect(normalizePhone(' (06) 11-00.00 01 ')).toBe('0611000001');
	});
});

describe('localeToBCP47', () => {
	it('maps app locales to Intl locales', () => {
		expect(localeToBCP47('en')).toBe('en-GB');
		expect(localeToBCP47('fr')).toBe('fr-FR');
		expect(localeToBCP47('de')).toBe('de-DE');
	});
});

describe('flexLabel', () => {
	it('maps 0/30/60 to compact labels', () => {
		expect(flexLabel(0)).toMatch(/Exact/i);
		expect(flexLabel(30)).toContain('30');
		expect(flexLabel(60)).toContain('60');
	});
});

describe('defaultDeparture', () => {
	it('returns now + 1h rounded up to 5 min as a local datetime-local string', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2030-12-01T09:02:00'));
		// 09:02 + 1h = 10:02 → rounded up to 10:05
		expect(defaultDeparture()).toBe('2030-12-01T10:05');
	});
});
```

- [ ] **Step 3: Run it to confirm failure**

Run: `cd frontend && npm run test:unit -- --run src/lib/utils.test.ts`
Expected: FAIL (functions not exported).

- [ ] **Step 4: Implement the utils (append below the shadcn `cn` export)**

Append to `frontend/src/lib/utils.ts`:
```typescript
import { getLocale } from '$lib/paraglide/runtime';
import { m } from '$lib/paraglide/messages';
import type { Flexibility } from './types';

const BCP47: Record<string, string> = {
	fr: 'fr-FR', en: 'en-GB', es: 'es-ES', it: 'it-IT', de: 'de-DE', nl: 'nl-NL'
};

export function localeToBCP47(locale: string = getLocale()): string {
	return BCP47[locale] ?? 'en-GB';
}

export function normalizePhone(phone: string): string {
	return phone.trim().replace(/[\s.\-()]/g, '');
}

/** Compact flexibility tag, e.g. "Exact" / "±30 min" / "±60 min". */
export function flexLabel(flex: Flexibility | number): string {
	if (flex === 30) return m.flexLabel30();
	if (flex === 60) return m.flexLabel60();
	if (flex === 0) return m.flexLabelExact();
	return `${flex} min`;
}

function pad(n: number): string {
	return String(n).padStart(2, '0');
}

/** Local now + 1h, rounded up to the next 5 minutes, as a `datetime-local` value. */
export function defaultDeparture(): string {
	const d = new Date(Date.now() + 60 * 60 * 1000);
	const rem = d.getMinutes() % 5;
	if (rem !== 0) d.setMinutes(d.getMinutes() + (5 - rem), 0, 0);
	d.setSeconds(0, 0);
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** "Wed 4 Jun at 14:30" in the active locale. */
export function formatTime(iso: string): string {
	const loc = localeToBCP47();
	const d = new Date(iso);
	const date = d.toLocaleDateString(loc, { weekday: 'short', day: 'numeric', month: 'short' });
	const time = d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' });
	return `${date} ${m.at()} ${time}`;
}

/** "Wed 4 Jun" in the active locale. */
export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString(localeToBCP47(), {
		weekday: 'short', day: 'numeric', month: 'short'
	});
}
```

- [ ] **Step 5: Run the test to confirm pass**

Run: `cd frontend && npm run test:unit -- --run src/lib/utils.test.ts`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): utils (format, phone, flex, locale map) + tests"
```

---

### Task 8: `stores.ts` — persisted profile + last search

**Files:**
- Create: `frontend/src/lib/stores.ts`
- Test: `frontend/src/lib/stores.test.ts`

- [ ] **Step 1: Install the persisted-store helper**

Run: `cd frontend && npm i @friendofsvelte/typed-local-store` — **OR** the spec's choice `npm i svelte-persisted-store`. This plan uses `svelte-persisted-store` (`persisted(key, initial)`); commands below assume it.
Run: `cd frontend && npm i svelte-persisted-store`

- [ ] **Step 2: Write the failing test**

`frontend/src/lib/stores.test.ts`:
```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { userName, userPhone, lastOrigin, lastDestination } from './stores';

describe('stores', () => {
	beforeEach(() => localStorage.clear());

	it('persists userName to the user_name key', () => {
		userName.set('Marie');
		expect(localStorage.getItem('user_name')).toBe(JSON.stringify('Marie'));
		expect(get(userName)).toBe('Marie');
	});

	it('persists userPhone to the user_phone key', () => {
		userPhone.set('0644000001');
		expect(localStorage.getItem('user_phone')).toBe(JSON.stringify('0644000001'));
	});

	it('defaults last-search fields to empty strings', () => {
		expect(get(lastOrigin)).toBe('');
		expect(get(lastDestination)).toBe('');
	});
});
```
> Note: `svelte-persisted-store` JSON-encodes values, so `user_name` holds `"Marie"` (quoted). Task 30 updates the e2e helper that seeds these keys to match (`localStorage.setItem('user_name', JSON.stringify('Marie'))`), and test 30's assertion reads through the same encoding.

- [ ] **Step 3: Run it to confirm failure**

Run: `cd frontend && npm run test:unit -- --run src/lib/stores.test.ts`
Expected: FAIL (module missing).

- [ ] **Step 4: Implement the stores**

`frontend/src/lib/stores.ts`:
```typescript
import { persisted } from 'svelte-persisted-store';

// Keys preserved verbatim from the legacy app for user-data continuity (Appendix D).
export const userName = persisted<string>('user_name', '');
export const userPhone = persisted<string>('user_phone', '');
export const lastOrigin = persisted<string>('last_origin', '');
export const lastDestination = persisted<string>('last_destination', '');
```

- [ ] **Step 5: Run the test to confirm pass**

Run: `cd frontend && npm run test:unit -- --run src/lib/stores.test.ts`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): persisted stores (profile, last search)"
```

---

### Task 9a: Backend — OpenAPI (`swaggo/swag`) annotations + `make swagger`

> Mirrors `/home/zeno/bizniz/dev/bizbiz-apiserver`. Annotate every Gin handler so `swag init` emits an accurate `docs/swagger.json` the frontend generates from. The schema is derived from the actual Go structs, so the inconsistent casing (PascalCase rides/requests, snake_case interests/notifications, camelCase config/vapid) is captured automatically — that is the whole point.

**Files:**
- Modify: `main.go` (general API info annotations above `func main()`)
- Modify: every handler in `internal/boundaries/handler/*.go` (swag comment block per handler func)
- Create (if needed): named response DTO structs for handlers that currently return `gin.H`/inline objects, so swag can name + shape them (e.g. `ContactInfo`, `ExpressInterestResponse`, `AcceptInterestResponse`, `ConfigResponse`, `VapidKeyResponse`, `ErrorResponse`)
- Create (generated): `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- Modify: `go.mod` (`github.com/swaggo/swag`), `Makefile` (`swagger`, `swagger-install` targets)

- [ ] **Step 1: Study the reference annotation style**

Read 1–2 annotated handlers in `/home/zeno/bizniz/dev/bizbiz-apiserver/internal/interfaces/webservice/gin/handlers_authuser.go` (the `@Summary`/`@Tags`/`@Param`/`@Success {object} T`/`@Failure`/`@Router` block above each func). Read `/home/zeno/bizniz/dev/bizbiz-apiserver/Makefile` `swagger`/`swagger-install` targets.

- [ ] **Step 2: General info in `main.go`**

Above `func main()`:
```go
// @title        Go-Stop API
// @version      1.0
// @description  Local ride-sharing notice board API.
// @BasePath     /api
```

- [ ] **Step 3: Annotate every handler.** First enumerate the routes by reading the `api := r.Group("/api")` block in `main.go`, then annotate each handler function. Use these stable `@ID`s (they become the generated TS function names — keep them EXACT):

| Method & path (under `/api`) | `@ID` | `@Tags` | Request | Success |
|---|---|---|---|---|
| GET `/rides` | `listRides` | rides | query: origin, destination, departure_at, search_date, search_time; header `X-Phone` (optional) | 200 `{array}` publicRide (or full Ride[] in my-rides mode — annotate the public shape) |
| POST `/rides` | `createRide` | rides | body PostRideBody | 201 `{object}` Ride |
| GET `/rides/{id}` | `getRide` | rides | path id | 200 `{object}` Ride |
| DELETE `/rides/{id}` | `deleteRide` | rides | path id; body `{phone}` | 204 |
| GET `/rides/{id}/interests` | `listRideInterests` | interests | path id; header X-Phone | 200 `{array}` InterestListItem |
| GET `/rides/{id}/requests` | `listRideRequests` | rides | path id; header X-Phone | 200 `{array}` PublicRequest |
| POST `/rides/{id}/interest` | `expressInterest` | interests | path id; body `{phone,name}` | 201 `{object}` ExpressInterestResponse |
| POST `/rides/{id}/feedback` | `submitRideFeedback` | rides | path id; body `{phone,taken}` | 204 |
| POST `/interests/{id}/accept` | `acceptInterest` | interests | path id; body `{phone}` | 200 `{object}` AcceptInterestResponse |
| GET `/interests` | `listMyInterests` | interests | header X-Phone | 200 `{array}` MyInterest |
| GET `/interests/{id}/contact` | `getInterestContact` | interests | path id; header X-Phone | 200 `{object}` ContactInfo |
| POST `/requests` | `createRequest` | requests | body PostRequestBody | 201 `{object}` Request |
| GET `/requests` | `listRequests` | requests | header X-Phone | 200 `{array}` Request |
| GET `/requests/{id}` | `getRequest` | requests | path id; header X-Phone | 200 `{object}` Request |
| DELETE `/requests/{id}` | `deleteRequest` | requests | path id; body `{phone}` | 204 |
| POST `/requests/{id}/ping` | `pingRequest` | requests | path id; header X-Phone; body `{ride_id}` | 204 |
| POST `/subscriptions` | `upsertSubscription` | subscriptions | body SubscriptionBody | 201 |
| DELETE `/subscriptions/{phone}` | `removeSubscription` | subscriptions | path phone | 204 |
| GET `/notifications` | `listNotifications` | notifications | header X-Phone | 200 `{array}` NotificationItem |
| GET `/config` | `getConfig` | config | — | 200 `{object}` ConfigResponse |
| GET `/stats` | `getStats` | stats | — | 200 `{object}` Stats |
| GET `/vapid-public-key` | `getVapidPublicKey` | vapid | — | 200 `{object}` VapidKeyResponse |
| GET `/destinations` | `listDestinations` | destinations | — | 200 `{array}` string |

Example block (adapt per handler; `@Router` path is relative to `@BasePath /api`):
```go
// DeleteRide deletes a ride owned by the caller.
// @ID       deleteRide
// @Tags     rides
// @Accept   json
// @Param    id    path  string                 true  "Ride ID"
// @Param    body  body  handler.DeleteRideBody true  "Owner phone"
// @Success  204
// @Failure  400  {object}  handler.ErrorResponse
// @Failure  403  {object}  handler.ErrorResponse
// @Router   /rides/{id} [delete]
func (h *RideHandler) Delete(c *gin.Context) { ... }
```
For handlers that bind an inline anonymous body or return `gin.H`, introduce a named struct in the handler package (e.g. `type DeleteRideBody struct { Phone string \`json:"phone"\` }`, `type ErrorResponse struct { Error string \`json:"error"\` }`, `type ContactInfo struct { Phone string \`json:"phone"\`; Name string \`json:"name"\`; Role string \`json:"role"\`; Origin string \`json:"origin"\`; Destination string \`json:"destination"\`; DepartureAt string \`json:"departure_at"\` }`, etc.) and reference it in the annotation. Where the handler returns a domain struct directly (`domain.Ride`, `domain.Request` — PascalCase, no json tags), reference that type so the generated schema is PascalCase (matching the wire format). Where it returns the handler-local `publicRide`/`publicRequest`, reference those (export them if swag can't resolve unexported types).

- [ ] **Step 4: Makefile targets**

Append to `/home/zeno/dev/go-stop/Makefile` (tabs):
```makefile
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

swagger:
	swag init -g main.go -o docs --parseDependency --parseInternal
```

- [ ] **Step 5: Generate the spec**

Run: `cd /home/zeno/dev/go-stop && (command -v swag || make swagger-install) && make swagger`
Expected: `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go` created. `go build ./...` still succeeds (docs.go compiles).

- [ ] **Step 6: Verify the spec reflects real casing**

Inspect `docs/swagger.json`: the `Ride`/`domain.Ride` schema must have PascalCase properties (`ID`, `DriverName`, `Phone`, `Origin`, `Destination`, `DepartureAt`, `Flexibility`, …); the interest/notification/contact schemas snake_case (`searcher_name`, `ride_id`, `departure_at`); config/vapid camelCase (`siteName`, `returnDelayHours`, `publicKey`). If any are wrong, the struct/annotation needs the right `json` tag or DTO. (Note: `swag` uses the `json` tag when present, else the Go field name.)

- [ ] **Step 7: Commit**

```bash
cd /home/zeno/dev/go-stop
go mod tidy
git add main.go internal/ docs/ Makefile go.mod go.sum
git commit -m "feat(server): OpenAPI (swag) annotations + make swagger; generate docs/swagger.json"
```

---

### Task 9b: Frontend — generate the API client + types via `orval`, facade, tests

> Mirrors `/home/zeno/bizniz/dev/bizniz-react-frontend` (`orval.config.ts`, single generated file, custom mutator, consumed via an adapter). The generated file + spec are COMMITTED so the production build needs no codegen step. The hand-written facade (`api.ts`) and `types.ts` re-exports give downstream Tasks 13–28 the exact `api.<resource>.<verb>()` surface and friendly type names from **Appendix A** (the FACADE CONTRACT).

**Files:**
- Modify: `frontend/package.json` (orval devDep + `api:generate` script)
- Create: `frontend/orval.config.ts`
- Create: `frontend/src/lib/api/fetchMutator.ts`
- Create (generated, committed): `frontend/src/lib/api/generated/go-stop-api.ts`
- Create: `frontend/src/lib/api.ts` (facade), update `frontend/src/lib/types.ts` (re-exports)
- Test: `frontend/src/lib/api.test.ts`
- Modify: `Makefile` (root `api-generate` target), `.gitignore` (do NOT ignore the generated client — it is committed)

- [ ] **Step 1: Install orval + add scripts**

`cd frontend && npm i -D orval`. Add to `frontend/package.json` scripts: `"api:generate": "orval --config ./orval.config.ts"`.

- [ ] **Step 2: `frontend/orval.config.ts`** (input = the Task 9a spec; single file; fetch client + custom mutator)
```typescript
import { defineConfig } from 'orval';

export default defineConfig({
	gostop: {
		input: { target: '../docs/swagger.json' },
		output: {
			target: './src/lib/api/generated/go-stop-api.ts',
			mode: 'single',
			client: 'fetch',
			override: {
				mutator: { path: './src/lib/api/fetchMutator.ts', name: 'customFetch' }
			}
		}
	}
});
```
> If the installed `orval` version's `fetch` client + mutator integration proves awkward, the documented fallback is `client: 'axios-functions'` with an axios mutator (exactly the bizniz setup: `npm i axios`, mutator returns an axios call). Prefer `fetch` (no extra dep, same-origin `/api`); only fall back if needed, and note it.

- [ ] **Step 3: `frontend/src/lib/api/fetchMutator.ts`** — base URL, error throwing, 204→null. Exact behavior the facade contract needs:
```typescript
export class ApiError extends Error {
	constructor(public status: number, message: string) {
		super(message);
		this.name = 'ApiError';
	}
}

// orval's fetch client calls customFetch(url, init) and expects the parsed body.
export async function customFetch<T>(url: string, init?: RequestInit): Promise<T> {
	const res = await fetch(url, init);
	if (res.status === 204) return null as T;
	const text = await res.text();
	const data = text ? JSON.parse(text) : null;
	if (!res.ok) {
		const msg = data && typeof data === 'object' && 'error' in data ? (data as { error: string }).error : res.statusText;
		throw new ApiError(res.status, msg);
	}
	return data as T;
}
```
> The swagger `@BasePath /api` makes generated paths absolute `/api/...`; with the Vite dev proxy and same-origin prod this needs no base prefix. If orval emits paths WITHOUT the `/api` prefix, prepend it in the mutator (`fetch(\`/api${url}\`, init)`). Verify against the generated output and adjust.

- [ ] **Step 4: Generate**

Run: `cd frontend && npm run api:generate` → creates `src/lib/api/generated/go-stop-api.ts` with one function per `@ID` (`listRides`, `createRide`, `deleteRide`, `getConfig`, …) and model types from the schemas. Inspect the generated function signatures + model names.

- [ ] **Step 5: `frontend/src/lib/types.ts`** — re-export the generated models under the friendly names the components use, and keep the `Flexibility` union. Adjust the right-hand generated names to whatever orval emitted (e.g. `DomainRide`, `HandlerPublicRide`, etc.):
```typescript
export type Flexibility = 0 | 30 | 60;
export type {
	DomainRide as Ride,
	HandlerPublicRide as PublicRide,
	HandlerPublicRequest as PublicRequest,
	DomainRequest as Request,
	HandlerInterestListItem as InterestListItem,
	HandlerMyInterest as MyInterest,
	HandlerContactInfo as ContactInfo,
	HandlerExpressInterestResponse as ExpressInterestResponse,
	HandlerAcceptInterestResponse as AcceptInterestResponse,
	HandlerNotificationItem as NotificationItem,
	DomainStats as Stats,
	HandlerConfigResponse as Config,
	HandlerVapidKeyResponse as VapidKey
	// ...map the rest; see Appendix A for the full set of friendly names
} from './api/generated/go-stop-api';
```
Keep request-body types (`PostRideBody`, `PostRequestBody`, `SubscriptionBody`, `RideSearchParams`) as friendly aliases of the generated param/body types, OR hand-define the few that orval inlines. The downstream contract is the Appendix A names.

- [ ] **Step 6: `frontend/src/lib/api.ts`** — the facade. Implement EXACTLY the `api` object surface from **Appendix A** (`api.rides.list/get/post/del/listInterests/listMatchingRequests/feedback`, `api.requests.list/get/post/del/ping`, `api.interests.express/accept/getContact/listMine`, `api.subscriptions.upsert/remove`, `api.notifications.list`, `api.config.get`, `api.stats.get`, `api.vapid.getPublicKey`, `api.destinations.list`), each delegating to the matching generated function. Map the X-Phone header arg and path/body params to the generated signatures. Re-export `ApiError` from the mutator. The point: downstream code keeps calling `api.rides.list(params, phone)` etc. unchanged.

- [ ] **Step 7: Tests** `frontend/src/lib/api.test.ts` — stub global `fetch` and assert the FACADE behavior (intent preserved from the original): `api.config.get()` resolves parsed JSON; a 403 `{error}` rejects with `ApiError` (status+message); 204 → `null`; an owner-scoped read sends the `X-Phone` header with the phone; a mutation sends `phone` in the JSON body; a search omits empty query params. Adapt the exact URL/header assertions to what the generated client + mutator actually produce (inspect a real call), keeping the behavioral intent. At least one test per: error-throwing, 204→null, X-Phone read, body mutation, query building.

- [ ] **Step 8: Run tests + check + build**

Run: `cd frontend && npx vitest run src/lib/api.test.ts` (PASS) and `npm run check` (no errors in api.ts/types.ts/mutator — ignore the known generated `paraglide/server.js` async_hooks error) and `npm run build` (succeeds; confirms `utils.ts`'s `Flexibility` import from `./types` still resolves).

- [ ] **Step 9: Root `Makefile` convenience target + commit**

Append to `/home/zeno/dev/go-stop/Makefile`:
```makefile
api-generate: swagger
	npm run api:generate --prefix frontend
```
Confirm the generated client is NOT gitignored (it is committed). Commit:
```bash
cd /home/zeno/dev/go-stop
git add frontend Makefile
git commit -m "feat(frontend): generate API client+types via orval; facade + tests"
```

---

### Task 10: `pwa.ts` — push, A2HS, polling, bell state

**Files:**
- Create: `frontend/src/lib/pwa.ts`
- Test: `frontend/src/lib/pwa.test.ts`

> Port of all push/A2HS/poll logic from `app.js`. The module exposes stores + functions; the modals/toasts are Svelte components (Task 18) that react to these stores. No direct DOM manipulation here.

- [ ] **Step 1: Write the failing test (pure/​testable surface)**

`frontend/src/lib/pwa.test.ts`:
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { urlBase64ToUint8Array, filterUnseenNotifications, maybeMarkStandalonePrompted } from './pwa';

beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());

describe('urlBase64ToUint8Array', () => {
	it('decodes a base64url VAPID key to bytes', () => {
		const out = urlBase64ToUint8Array('AQID'); // base64 for [1,2,3]
		expect(Array.from(out)).toEqual([1, 2, 3]);
	});
});

describe('filterUnseenNotifications', () => {
	it('returns only ride_ids not already in poll_seen and records them', () => {
		localStorage.setItem('poll_seen', JSON.stringify(['r1']));
		const incoming = [{ ride_id: 'r1' }, { ride_id: 'r2' }] as any;
		const fresh = filterUnseenNotifications(incoming);
		expect(fresh.map((n: any) => n.ride_id)).toEqual(['r2']);
		expect(JSON.parse(localStorage.getItem('poll_seen')!)).toContain('r2');
	});
});

describe('maybeMarkStandalonePrompted', () => {
	it('returns true once then false (sets the flag)', () => {
		expect(maybeMarkStandalonePrompted()).toBe(true);
		expect(maybeMarkStandalonePrompted()).toBe(false);
		expect(localStorage.getItem('standalone_notif_prompted')).toBe('1');
	});
});
```

- [ ] **Step 2: Run it to confirm failure**

Run: `cd frontend && npm run test:unit -- --run src/lib/pwa.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement `pwa.ts`**

`frontend/src/lib/pwa.ts`:
```typescript
import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { api } from './api';
import type { NotificationItem } from './types';

export type PushState = 'unsupported' | 'ios-browser' | 'default' | 'granted' | 'denied' | 'subscribed';

export const pushState = writable<PushState>('default');
/** Notifications surfaced by polling; the layout renders a PollToast per item. */
export const pollToasts = writable<NotificationItem[]>([]);

export function isIOSBrowser(): boolean {
	if (!browser) return false;
	const ua = navigator.userAgent;
	const isIOS = /iPad|iPhone|iPod/.test(ua) && !(window as unknown as { MSStream?: unknown }).MSStream;
	const standalone =
		(navigator as unknown as { standalone?: boolean }).standalone === true ||
		window.matchMedia('(display-mode: standalone)').matches;
	return isIOS && !standalone;
}

export function isStandalone(): boolean {
	if (!browser) return false;
	return (
		(navigator as unknown as { standalone?: boolean }).standalone === true ||
		window.matchMedia('(display-mode: standalone)').matches
	);
}

export function urlBase64ToUint8Array(base64String: string): Uint8Array {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const raw = atob(base64);
	const out = new Uint8Array(raw.length);
	for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
	return out;
}

/** Subscribe to Web Push and register with the backend. Returns success. */
export async function trySubscribePush(phone: string): Promise<boolean> {
	if (!browser || !('serviceWorker' in navigator) || !('PushManager' in window)) return false;
	try {
		const reg = await navigator.serviceWorker.ready;
		const { publicKey } = await api.vapid.getPublicKey();
		if (!publicKey) return false;
		const sub = await reg.pushManager.subscribe({
			userVisibleOnly: true,
			applicationServerKey: urlBase64ToUint8Array(publicKey)
		});
		const json = sub.toJSON() as { endpoint?: string; keys?: { p256dh?: string; auth?: string } };
		if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false;
		await api.subscriptions.upsert({ phone, endpoint: json.endpoint, p256dh: json.keys.p256dh, auth: json.keys.auth });
		return true;
	} catch {
		return false;
	}
}

/** Recompute the bell/push state; silently re-subscribe if a subscription expired. */
export async function updateBellState(phone?: string): Promise<void> {
	if (!browser) return;
	if (isIOSBrowser()) return pushState.set('ios-browser');
	if (!('Notification' in window) || !('serviceWorker' in navigator)) return pushState.set('unsupported');
	const perm = Notification.permission;
	if (perm === 'denied') return pushState.set('denied');
	if (perm !== 'granted') return pushState.set('default');
	const reg = await navigator.serviceWorker.ready;
	const sub = await reg.pushManager.getSubscription();
	if (sub) return pushState.set('subscribed');
	if (phone && (await trySubscribePush(phone))) return pushState.set('subscribed');
	pushState.set('granted');
}

const SEEN_KEY = 'poll_seen';

/** Pure: given incoming notifications, return those whose ride_id is unseen and record them (cap 100). */
export function filterUnseenNotifications(items: NotificationItem[]): NotificationItem[] {
	if (!browser) return items;
	let seen: string[] = [];
	try {
		seen = JSON.parse(localStorage.getItem(SEEN_KEY) ?? '[]');
	} catch {
		seen = [];
	}
	const fresh = items.filter((n) => !seen.includes(n.ride_id));
	if (fresh.length) {
		const next = [...seen, ...fresh.map((n) => n.ride_id)].slice(-100);
		localStorage.setItem(SEEN_KEY, JSON.stringify(next));
	}
	return fresh;
}

let lastPollMs = 0;

/** Poll the backend for ride-match notifications and push fresh ones to pollToasts (throttled 60s). */
export async function pollForNotifications(phone: string): Promise<void> {
	if (!browser || !phone) return;
	const now = Date.now();
	if (now - lastPollMs < 60_000) return;
	lastPollMs = now;
	try {
		const items = await api.notifications.list(phone);
		const fresh = filterUnseenNotifications(items);
		if (fresh.length) pollToasts.update((t) => [...t, ...fresh]);
	} catch {
		/* network errors are non-fatal for polling */
	}
}

/** Returns true the first time only (sets standalone_notif_prompted). */
export function maybeMarkStandalonePrompted(): boolean {
	if (!browser) return false;
	if (localStorage.getItem('standalone_notif_prompted')) return false;
	localStorage.setItem('standalone_notif_prompted', '1');
	return true;
}
```

- [ ] **Step 4: Run the test to confirm pass**

Run: `cd frontend && npm run test:unit -- --run src/lib/pwa.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Run the full unit suite**

Run: `cd frontend && npm run test:unit -- --run`
Expected: all Phase-1 tests pass.

- [ ] **Step 6: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): pwa module (push, A2HS, polling, bell state) + tests"
```

---

# Phase 2 — Layout & components (TDD)

Outcome of Phase 2: all shared components exist and are unit-tested. Routes (Phase 3) only wire them. Each component declares the Test-Hook Contract entries it owns (Appendix C).

> **Component test pattern (Svelte 5 + @testing-library/svelte):**
> ```typescript
> import { render, screen } from '@testing-library/svelte';
> import Comp from './Comp.svelte';
> // render(Comp, { props: {...} }); then assert via screen / container.
> ```
> `m.*()` works in tests because `pretest:unit` compiles Paraglide. Locale defaults to `fr` (baseLocale) unless a test sets `localStorage.lang` then `registerLangStrategy()`.

### Task 11: App shell — `+layout.svelte`, `TopBar`, `LangPicker`, config/title, footer

**Files:**
- Create: `frontend/src/routes/+layout.svelte`
- Create: `frontend/src/lib/components/layout/TopBar.svelte`
- Create: `frontend/src/lib/components/layout/LangPicker.svelte`
- Create: `frontend/src/lib/config.ts` (config store loaded once)
- Test: `frontend/src/lib/components/layout/LangPicker.test.ts`

**Test-hooks owned:** `<div id="app">` wrapper around page content; footer link text `Statistiques` (and Privacy); `#btn-me` (in TopBar).

- [ ] **Step 1: Config store**

`frontend/src/lib/config.ts`:
```typescript
import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { api } from './api';
import type { Config } from './types';

export const config = writable<Config>({ siteName: 'Go-Stop', returnDelayHours: 2 });

export async function loadConfig(): Promise<void> {
	if (!browser) return;
	try {
		const c = await api.config.get();
		config.set(c);
		document.title = `${c.siteName}`;
	} catch {
		/* keep defaults */
	}
}
```
> Note: the e2e test DB returns `siteName: 'Go Stop Saillans!'`; tests assert `document.title` and `<h1>` contain that exact string. The home `<h1>` (Task 19) and `document.title` both read `config.siteName`.

- [ ] **Step 2: LangPicker (failing test first)**

`frontend/src/lib/components/layout/LangPicker.test.ts`:
```typescript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import LangPicker from './LangPicker.svelte';

describe('LangPicker', () => {
	it('shows the current locale flag and a dropdown with all 6 locales', async () => {
		const { container } = render(LangPicker);
		// 6 language options exist (hidden until opened)
		expect(container.querySelectorAll('.lang-opt').length).toBe(6);
	});
});
```

- [ ] **Step 3: Run it to confirm failure**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/layout/LangPicker.test.ts`
Expected: FAIL.

- [ ] **Step 4: Implement LangPicker**

`frontend/src/lib/components/layout/LangPicker.svelte`:
```svelte
<script lang="ts">
	import { getLocale, setLocale, locales } from '$lib/locale';

	const FLAGS: Record<string, string> = { fr: '🇫🇷', en: '🇬🇧', es: '🇪🇸', it: '🇮🇹', de: '🇩🇪', nl: '🇳🇱' };
	const ORDER = ['fr', 'en', 'es', 'it', 'de', 'nl'] as const;
	let open = $state(false);
	let current = $derived(getLocale());

	function pick(code: string) {
		open = false;
		setLocale(code as never); // persists to localStorage["lang"] and reloads
	}
</script>

<div class="lang-picker relative">
	<button type="button" class="btn-lang" aria-label="Language" onclick={() => (open = !open)}>
		{FLAGS[current] ?? '🌐'}
	</button>
	{#if open}
		<div class="lang-dropdown absolute right-0 z-50 mt-1 rounded border bg-white shadow">
			{#each ORDER as code}
				<button type="button" class="lang-opt block w-full px-3 py-1 text-left" data-lang={code} onclick={() => pick(code)}>
					{FLAGS[code]} {code.toUpperCase()}
				</button>
			{:else}
				<!-- locales is the source of truth; ORDER must match it -->
			{/each}
		</div>
	{/if}
	<span class="hidden">{locales.length}</span>
</div>
```
> The `{#each ORDER}` always renders 6; `locales` import is referenced to keep parity with `project.inlang`. If `ORDER` and `locales` ever diverge, update `ORDER`.

- [ ] **Step 5: TopBar (controls cluster)**

`frontend/src/lib/components/layout/TopBar.svelte`:
```svelte
<script lang="ts">
	import LangPicker from './LangPicker.svelte';
	import BellButton from '$lib/components/notifications/BellButton.svelte';
	import { userName } from '$lib/stores';

	let { onabout, onprivacy }: { onabout?: () => void; onprivacy?: () => void } = $props();
	let hasProfile = $derived($userName.length > 0);
</script>

<div class="controls flex items-center gap-2">
	<LangPicker />
	<div class="controls-icons ml-auto flex items-center gap-2">
		<a id="btn-me" href="/me" class="me-icon" class:me-icon-set={hasProfile} aria-label="Profile" title="Me">👤</a>
		<button type="button" class="btn-about" aria-label="About" onclick={onabout}>ⓘ</button>
		<BellButton />
	</div>
</div>
```
> `BellButton` is created in Task 18; until then, create a minimal stub `BellButton.svelte` exporting an empty `<span class="btn-bell"></span>` so TopBar compiles, and flesh it out in Task 18. (Add the stub now.)

- [ ] **Step 6: Layout shell**

`frontend/src/routes/+layout.svelte`:
```svelte
<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { registerLangStrategy } from '$lib/locale';
	import { loadConfig } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { get } from 'svelte/store';
	import { updateBellState, pollForNotifications, isStandalone, maybeMarkStandalonePrompted } from '$lib/pwa';
	import TopBar from '$lib/components/layout/TopBar.svelte';
	import AboutModal from '$lib/components/layout/AboutModal.svelte';
	import PrivacyModal from '$lib/components/layout/PrivacyModal.svelte';
	import A2HSBanner from '$lib/components/notifications/A2HSBanner.svelte';
	import PollToastHost from '$lib/components/notifications/PollToast.svelte';
	import NotifModal from '$lib/components/notifications/NotifModal.svelte';
	import { m } from '$lib/paraglide/messages';

	let { children } = $props();
	let showAbout = $state(false);
	let showPrivacy = $state(false);
	let isHome = $derived(page.url.pathname === '/');

	function back() {
		if (browser && history.length > 1) history.back();
		else goto('/');
	}

	if (browser) registerLangStrategy();

	onMount(() => {
		loadConfig();
		const phone = get(userPhone);
		updateBellState(phone);
		if (isStandalone() && maybeMarkStandalonePrompted()) {
			// NotifModal opens itself via pushState (Task 18)
		}
		const onVis = () => {
			if (document.visibilityState === 'visible') pollForNotifications(get(userPhone));
		};
		document.addEventListener('visibilitychange', onVis);
		return () => document.removeEventListener('visibilitychange', onVis);
	});
</script>

<header class="top-bar mx-auto flex max-w-xl items-center gap-2 p-3" class:page-bar={!isHome}>
	{#if !isHome}
		<button id="back" type="button" class="btn-back" onclick={back}>{m.btnBack()}</button>
	{/if}
	<TopBar onabout={() => (showAbout = true)} onprivacy={() => (showPrivacy = true)} />
</header>

<div id="app" class="mx-auto max-w-xl p-3">
	{@render children()}
</div>

<footer id="app-footer" class="mx-auto max-w-xl p-3 text-center text-sm text-gray-500">
	<button type="button" class="btn-footer-privacy underline" onclick={() => (showPrivacy = true)}>{m.footerPrivacy()}</button>
	<span> · </span>
	<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
</footer>

<A2HSBanner onopen={() => (/* A2HSModal handled in Task 18 */ undefined)} />
<PollToastHost />
<NotifModal />
{#if showAbout}<AboutModal onclose={() => (showAbout = false)} />{/if}
{#if showPrivacy}<PrivacyModal onclose={() => (showPrivacy = false)} />{/if}
```
> The footer renders the stats link with text `m.statsPageTitle()` = "Statistiques" (fr), satisfying test 2's `getByText('Statistiques')`. Modal/banner components are built in Tasks 12 & 18; add minimal stubs now (empty components) so the layout compiles, then fill them in.

- [ ] **Step 7: Add compile-stubs for not-yet-built components**

Create empty-but-valid stubs so the project compiles after this task (each later task replaces its stub):
`AboutModal.svelte`, `PrivacyModal.svelte` (Task 12); `BellButton.svelte`, `NotifModal.svelte`, `A2HSBanner.svelte`, `PollToast.svelte` (Task 18). Minimal stub example for each:
```svelte
<script lang="ts">
	let { onclose, onopen }: { onclose?: () => void; onopen?: () => void } = $props();
</script>
```

- [ ] **Step 8: Run the LangPicker test + build**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/layout/LangPicker.test.ts && npm run build`
Expected: test passes; build succeeds.

- [ ] **Step 9: Commit**

```bash
cd /home/zeno/dev/go-stop
git add frontend
git commit -m "feat(frontend): app shell (layout, TopBar, LangPicker, config, footer)"
```

---

### Task 12: About & Privacy modals, BackButton text

**Files:**
- Create: `frontend/src/lib/components/layout/AboutModal.svelte` (replace stub)
- Create: `frontend/src/lib/components/layout/PrivacyModal.svelte` (replace stub)
- Test: `frontend/src/lib/components/layout/PrivacyModal.test.ts`

- [ ] **Step 1: Failing test**

`frontend/src/lib/components/layout/PrivacyModal.test.ts`:
```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import PrivacyModal from './PrivacyModal.svelte';

describe('PrivacyModal', () => {
	it('renders the privacy heading and a close button', async () => {
		const onclose = vi.fn();
		render(PrivacyModal, { props: { onclose } });
		expect(screen.getByText(/Confidentialité|Privacy/)).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /Fermer|Close/ })).toBeInTheDocument();
	});
});
```

- [ ] **Step 2: Run → FAIL**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/layout/PrivacyModal.test.ts`

- [ ] **Step 3: Implement modals (use shadcn Dialog or a plain overlay)**

`frontend/src/lib/components/layout/PrivacyModal.svelte`:
```svelte
<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	let { onclose }: { onclose: () => void } = $props();
</script>

<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={onclose} role="presentation">
	<div class="modal max-h-[85vh] w-full max-w-lg overflow-auto rounded bg-white p-5" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
		<h3 class="mb-2 text-lg font-semibold">{m.privacyTitle()}</h3>
		<div class="prose prose-sm">{@html m.privacyBody()}</div>
		<button type="button" class="btn-modal-close mt-4 rounded border px-3 py-1" onclick={onclose}>{m.privacyClose()}</button>
	</div>
</div>
```

`frontend/src/lib/components/layout/AboutModal.svelte` (same shape; uses `aboutTitle`, `aboutBody({ siteName })`, and `privacyClose` for the close label):
```svelte
<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
	let { onclose }: { onclose: () => void } = $props();
</script>

<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={onclose} role="presentation">
	<div class="modal max-h-[85vh] w-full max-w-lg overflow-auto rounded bg-white p-5" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
		<h3 class="mb-2 text-lg font-semibold">{m.aboutTitle()}</h3>
		<div class="prose prose-sm">{@html m.aboutBody({ siteName: $config.siteName })}</div>
		<button type="button" class="btn-modal-close mt-4 rounded border px-3 py-1" onclick={onclose}>{m.privacyClose()}</button>
	</div>
</div>
```

- [ ] **Step 4: Run → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/layout/PrivacyModal.test.ts`

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): About & Privacy modals"
```

---

### Task 13: `ContactOrInterest` + `RideCard`

**Files:**
- Create: `frontend/src/lib/notifModal.ts` (shared open-state for NotifModal)
- Create: `frontend/src/lib/components/rides/ContactOrInterest.svelte`
- Create: `frontend/src/lib/components/rides/RideCard.svelte`
- Test: `frontend/src/lib/components/rides/ContactOrInterest.test.ts`

**Test-hooks owned:** `.btn-interest` (+ `data-ride-id`), `.interest-state`, `.card`, `.card-route`.

- [ ] **Step 1: NotifModal open-state module**

`frontend/src/lib/notifModal.ts`:
```typescript
import { writable } from 'svelte/store';
import type { PushState } from './pwa';

// null = closed; otherwise the state the modal should present.
export const notifModalState = writable<PushState | null>(null);
export function openNotifModal(state: PushState) {
	notifModalState.set(state);
}
```

- [ ] **Step 2: Failing test for the interest flow**

`frontend/src/lib/components/rides/ContactOrInterest.test.ts`:
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ContactOrInterest from './ContactOrInterest.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => { localStorage.clear(); userName.set('Bob'); userPhone.set('0622000002'); });
afterEach(() => vi.unstubAllGlobals());

const ride = { ID: '42', DriverName: 'Alice', Origin: 'Saillans', Destination: 'Crest', Flexibility: 0 } as any;

describe('ContactOrInterest', () => {
	it('renders a request-contact button with data-ride-id', () => {
		const { container } = render(ContactOrInterest, { props: { ride } });
		const btn = container.querySelector('.btn-interest') as HTMLElement;
		expect(btn).toBeTruthy();
		expect(btn.dataset.rideId).toBe('42');
	});

	it('expressing interest POSTs and stores interest_<id> in localStorage', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ id: 'int1', status: 'pending' }), { status: 201 })));
		const { container } = render(ContactOrInterest, { props: { ride } });
		await fireEvent.click(container.querySelector('.btn-interest')!);
		await vi.waitFor(() => expect(localStorage.getItem('interest_42')).toBe('int1'));
		expect(container.querySelector('.interest-state')!.textContent).toMatch(/envoyée|sent/i);
	});

	it('shows a tel link when a contact phone is already known', () => {
		const { container } = render(ContactOrInterest, { props: { ride, contactPhone: '0611000001' } });
		expect(container.querySelector('a[href="tel:0611000001"]')).toBeTruthy();
	});
});
```

- [ ] **Step 3: Run → FAIL**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/rides/ContactOrInterest.test.ts`

- [ ] **Step 4: Implement ContactOrInterest**

`frontend/src/lib/components/rides/ContactOrInterest.svelte`:
```svelte
<script lang="ts">
	import { browser } from '$app/environment';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { pushState, updateBellState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Ride } from '$lib/types';

	let { ride, contactPhone }: { ride: PublicRide | Ride; contactPhone?: string } = $props();

	const storedInterest = () => (browser ? localStorage.getItem(`interest_${ride.ID}`) : null);
	let pending = $state(!!storedInterest());
	let stateMsg = $state('');
	let busy = $state(false);

	async function express() {
		if (busy) return;
		busy = true;
		stateMsg = '';
		let phone = get(userPhone);
		if (!phone && browser) phone = window.prompt(m.labelPhone()) ?? '';
		if (!phone) { busy = false; return; }
		try {
			const res = await api.interests.express(ride.ID, phone, get(userName) || undefined);
			if (browser) localStorage.setItem(`interest_${ride.ID}`, res.id);
			pending = true;
			stateMsg = m.interestSent();
			if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		} catch (e) {
			stateMsg = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}
</script>

{#if contactPhone}
	<div class="contact-revealed">
		<a class="btn btn-primary" href="tel:{contactPhone}">{m.btnCallNow()}</a>
	</div>
{:else if pending}
	<div class="interest-pending-row flex items-center gap-2">
		<span class="interest-pending-label text-sm text-gray-600">{m.interestPending()}</span>
		<button type="button" class="btn-interest btn-interest-resend" data-ride-id={ride.ID} disabled={busy} onclick={express}>{m.btnResend()}</button>
		<span class="interest-state" id="int-state-{ride.ID}">{stateMsg}</span>
	</div>
{:else}
	<button type="button" class="btn-interest" data-ride-id={ride.ID} disabled={busy} onclick={express}>{m.btnInterest()}</button>
	<span class="interest-state" id="int-state-{ride.ID}">{stateMsg}</span>
{/if}
```

- [ ] **Step 5: Implement RideCard**

`frontend/src/lib/components/rides/RideCard.svelte`:
```svelte
<script lang="ts">
	import ContactOrInterest from './ContactOrInterest.svelte';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Ride } from '$lib/types';

	let {
		ride,
		contactPhone,
		showDriver = true
	}: { ride: PublicRide | Ride; contactPhone?: string; showDriver?: boolean } = $props();

	const interestCount = $derived('InterestCount' in ride ? ride.InterestCount : 0);
</script>

<div class="card card-compact rounded border p-3">
	<div class="card-route font-medium" translate="no">{ride.Origin} → {ride.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(ride.DepartureAt)}</span>
		<span class="tag rounded bg-gray-100 px-1">{flexLabel(ride.Flexibility)}</span>
		{#if showDriver && ride.DriverName}<span>{ride.DriverName}</span>{/if}
		{#if interestCount > 0}<span class="tag tag-interest-count rounded bg-blue-100 px-1">{m.interestCount({ count: interestCount })}</span>{/if}
	</div>
	<ContactOrInterest {ride} {contactPhone} />
</div>
```

- [ ] **Step 6: Run → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/rides/ContactOrInterest.test.ts`

- [ ] **Step 7: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): RideCard + ContactOrInterest (interest flow)"
```

---

### Task 14: `RideForm` — post-ride with return leg

**Files:**
- Create: `frontend/src/lib/components/rides/RideForm.svelte`
- Test: `frontend/src/lib/components/rides/RideForm.test.ts`

**Test-hooks owned:** input `name`s `driver_name`, `phone`, `origin`, `destination`, `departure_at`, `return_departure_at`; `#btn-return`; submit `button[type=submit].btn-primary`. (Test 10 asserts return time = outbound + `returnDelayHours`.)

- [ ] **Step 1: Failing test (return-leg default time)**

`frontend/src/lib/components/rides/RideForm.test.ts`:
```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import RideForm from './RideForm.svelte';
import { config } from '$lib/config';

beforeEach(() => { localStorage.clear(); config.set({ siteName: 'Go-Stop', returnDelayHours: 2 }); });

describe('RideForm', () => {
	it('return toggle defaults return time to outbound + returnDelayHours', async () => {
		const { container } = render(RideForm, { props: { destinations: [] } });
		const dep = container.querySelector('input[name=departure_at]') as HTMLInputElement;
		await fireEvent.input(dep, { target: { value: '2030-12-01T09:00' } });
		await fireEvent.click(container.querySelector('#btn-return')!);
		const ret = container.querySelector('input[name=return_departure_at]') as HTMLInputElement;
		expect(ret.value).toBe('2030-12-01T11:00');
	});
});
```

- [ ] **Step 2: Run → FAIL**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/rides/RideForm.test.ts`

- [ ] **Step 3: Implement RideForm**

`frontend/src/lib/components/rides/RideForm.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { config } from '$lib/config';
	import { defaultDeparture, normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility } from '$lib/types';

	let { destinations = [], onposted }: { destinations?: string[]; onposted?: (phone: string) => void } = $props();

	let driver_name = $state(get(userName));
	let phone = $state(get(userPhone));
	let origin = $state('');
	let destination = $state('');
	let departure_at = $state(defaultDeparture());
	let flexibility = $state<Flexibility>(30);
	let isReturn = $state(false);
	let return_departure_at = $state('');
	let return_flexibility = $state<Flexibility>(30);
	let err = $state('');

	function toggleReturn(on: boolean) {
		isReturn = on;
		if (on && !return_departure_at) {
			if (departure_at) {
				const d = new Date(departure_at);
				d.setHours(d.getHours() + $config.returnDelayHours);
				const p = (n: number) => String(n).padStart(2, '0');
				return_departure_at = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
			} else {
				return_departure_at = defaultDeparture();
			}
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		err = '';
		const ph = normalizePhone(phone);
		userName.set(driver_name);
		userPhone.set(ph);
		try {
			await api.rides.post({
				driver_name, phone: ph, origin, destination,
				departure_at: new Date(departure_at).toISOString(), flexibility
			});
			if (isReturn && return_departure_at) {
				await api.rides.post({
					driver_name, phone: ph, origin: destination, destination: origin,
					departure_at: new Date(return_departure_at).toISOString(), flexibility: return_flexibility
				});
			}
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
</script>

<form id="ride-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelName()}<input name="driver_name" required bind:value={driver_name} /></label>
	<label>{m.labelPhone()}<input name="phone" type="tel" required bind:value={phone} /></label>
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={origin} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destination} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<label>{m.labelDatetime()}<input name="departure_at" type="datetime-local" step="300" required bind:value={departure_at} /></label>
	<label>{m.labelFlex()}
		<select bind:value={flexibility}>
			<option value={0}>{m.flexExact()}</option>
			<option value={30}>{m.flex30()}</option>
			<option value={60}>{m.flex60()}</option>
		</select>
	</label>

	<div class="trip-type-toggle flex gap-2" role="group" aria-label={m.tripTypeLabel()}>
		<button id="btn-oneway" type="button" class:active={!isReturn} onclick={() => toggleReturn(false)}>{m.tripOneWay()}</button>
		<button id="btn-return" type="button" class:active={isReturn} onclick={() => toggleReturn(true)}>{m.tripReturn()}</button>
	</div>

	{#if isReturn}
		<fieldset id="return-section" class="return-section flex flex-col gap-2 rounded border p-2">
			<legend>{m.returnSection()}</legend>
			<label>{m.labelReturnTime()}<input name="return_departure_at" type="datetime-local" step="300" bind:value={return_departure_at} required={isReturn} /></label>
			<label>{m.labelReturnFlex()}
				<select bind:value={return_flexibility}>
					<option value={0}>{m.flexExact()}</option>
					<option value={30}>{m.flex30()}</option>
					<option value={60}>{m.flex60()}</option>
				</select>
			</label>
		</fieldset>
	{/if}

	<button type="submit" class="btn btn-primary">{m.btnPostRide()}</button>
	{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
</form>
```

- [ ] **Step 4: Run → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/rides/RideForm.test.ts`

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): RideForm with return-leg toggle"
```

---

### Task 15: `SeekerRow` — driver view of a matching searcher + Ping

**Files:**
- Create: `frontend/src/lib/components/rides/SeekerRow.svelte`
- Test: `frontend/src/lib/components/rides/SeekerRow.test.ts`

**Test-hooks owned:** `.seeker-row`; `.btn-ping-searcher` (+ `data-req-id`, `data-ride-id`). (Test 29: shows searcher name not phone; click → `POST .../ping` 204 → button disables/✓.)

- [ ] **Step 1: Failing test**

`frontend/src/lib/components/rides/SeekerRow.test.ts`:
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SeekerRow from './SeekerRow.svelte';

afterEach(() => vi.unstubAllGlobals());
const req = { ID: 'rq1', SearcherName: 'Bob', Origin: 'Saillans', Destination: 'Crest', DepartureAt: '2030-06-15T08:00:00Z', Flexibility: 0 } as any;

describe('SeekerRow', () => {
	it('shows the searcher name, carries data attrs, and pings on click', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response(null, { status: 204 })));
		const { container } = render(SeekerRow, { props: { request: req, rideId: 'r9', driverPhone: '0611000001' } });
		const btn = container.querySelector('.btn-ping-searcher') as HTMLButtonElement;
		expect(container.querySelector('.seeker-row')!.textContent).toContain('Bob');
		expect(btn.dataset.reqId).toBe('rq1');
		expect(btn.dataset.rideId).toBe('r9');
		await fireEvent.click(btn);
		await vi.waitFor(() => expect(btn.disabled).toBe(true));
	});
});
```

- [ ] **Step 2: Run → FAIL**, then implement.

`frontend/src/lib/components/rides/SeekerRow.svelte`:
```svelte
<script lang="ts">
	import { api } from '$lib/api';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRequest } from '$lib/types';

	let { request, rideId, driverPhone }: { request: PublicRequest; rideId: string; driverPhone: string } = $props();
	let done = $state(false);
	let busy = $state(false);

	async function ping() {
		if (busy || done) return;
		busy = true;
		try {
			await api.requests.ping(request.ID, rideId, driverPhone);
			done = true;
		} finally {
			busy = false;
		}
	}
</script>

<div class="seeker-row flex items-center gap-2">
	<div class="grow">
		<div>{request.SearcherName}</div>
		<div class="text-sm text-gray-600">{formatTime(request.DepartureAt)} · {flexLabel(request.Flexibility)}</div>
	</div>
	<button type="button" class="btn-ping-searcher" data-req-id={request.ID} data-ride-id={rideId} disabled={busy || done} onclick={ping}>
		{done ? '✓' : m.btnPingSearcher()}
	</button>
</div>
```

- [ ] **Step 3: Run → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/rides/SeekerRow.test.ts`

- [ ] **Step 4: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): SeekerRow + ping"
```

---

### Task 16: `AlertForm` + `AlertCard`

**Files:**
- Create: `frontend/src/lib/components/alerts/AlertForm.svelte`
- Create: `frontend/src/lib/components/alerts/AlertCard.svelte`
- Test: `frontend/src/lib/components/alerts/AlertForm.test.ts`

**Test-hooks owned (AlertForm):** `#notify-form`; input `name`s `searcher_name`, `phone`, `origin`, `destination`, `alert_date`, `alert_time`; `.btn-mode` (+ `data-mode="time|day|daily|anytime"`); submit. **(AlertCard):** `.card`, `.btn-see-matches` (+ `data-origin`,`data-dest`,`data-dept`), `.btn-delete` (+ `data-id`,`data-phone`), `.delete-msg`.

- [ ] **Step 1: Failing test (mode=time with date+time posts departure_at)**

`frontend/src/lib/components/alerts/AlertForm.test.ts`:
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AlertForm from './AlertForm.svelte';

beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());

describe('AlertForm', () => {
	it('renders #notify-form with 4 mode buttons', () => {
		const { container } = render(AlertForm, { props: { origin: 'Saillans', destination: 'Crest' } });
		expect(container.querySelector('#notify-form')).toBeTruthy();
		expect(container.querySelectorAll('.btn-mode').length).toBe(4);
	});

	it('mode=time with date+time posts a departure_at instant', async () => {
		let body: any;
		vi.stubGlobal('fetch', vi.fn(async (_u: string, init: RequestInit) => { body = JSON.parse(init.body as string); return new Response(JSON.stringify({ ID: 'x' }), { status: 201 }); }));
		const { container } = render(AlertForm, { props: { origin: 'Saillans', destination: 'Crest' } });
		await fireEvent.input(container.querySelector('input[name=searcher_name]')!, { target: { value: 'Bob' } });
		await fireEvent.input(container.querySelector('input[name=phone]')!, { target: { value: '0622000002' } });
		await fireEvent.input(container.querySelector('input[name=alert_date]')!, { target: { value: '2030-06-15' } });
		await fireEvent.input(container.querySelector('input[name=alert_time]')!, { target: { value: '09:30' } });
		await fireEvent.submit(container.querySelector('#notify-form')!);
		await vi.waitFor(() => expect(body).toBeTruthy());
		expect(body.departure_at).toBeTruthy();
		expect(body.searcher_name).toBe('Bob');
	});
});
```

- [ ] **Step 2: Run → FAIL**, then implement AlertForm.

`frontend/src/lib/components/alerts/AlertForm.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility, PostRequestBody } from '$lib/types';

	let {
		origin = '', destination = '', departureAt = '', destinations = [], onposted
	}: { origin?: string; destination?: string; departureAt?: string; destinations?: string[]; onposted?: (phone: string) => void } = $props();

	type Mode = 'time' | 'day' | 'daily' | 'anytime';
	let mode = $state<Mode>('time');
	let searcher_name = $state(get(userName));
	let phone = $state(get(userPhone));
	let originV = $state(origin);
	let destinationV = $state(destination);
	// Prefill date/time from a passed RFC3339 departureAt (search "notify" deep link).
	let alert_date = $state(departureAt ? departureAt.slice(0, 10) : '');
	let alert_time = $state(departureAt ? new Date(departureAt).toTimeString().slice(0, 5) : '');
	let flexibility = $state<Flexibility>(30);
	let err = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		err = '';
		const ph = normalizePhone(phone);
		userName.set(searcher_name);
		userPhone.set(ph);
		const body: PostRequestBody = { searcher_name, phone: ph, origin: originV, destination: destinationV };
		if (mode === 'time') {
			if (alert_date && alert_time) { body.departure_at = new Date(`${alert_date}T${alert_time}`).toISOString(); body.flexibility = flexibility; }
			else if (alert_date) body.departure_date = alert_date;
		} else if (mode === 'day') {
			if (alert_date) body.departure_date = alert_date;
		} else if (mode === 'daily') {
			if (alert_time) { body.departure_time = alert_time; body.flexibility = flexibility; }
		} // anytime → nothing
		try {
			await api.requests.post(body);
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
	const modes: { key: Mode; label: () => string }[] = [
		{ key: 'time', label: m.alertModeTime }, { key: 'day', label: m.alertModeDay },
		{ key: 'daily', label: m.alertModeDaily }, { key: 'anytime', label: m.alertModeAnytime }
	];
</script>

<form id="notify-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelName()}<input name="searcher_name" required bind:value={searcher_name} /></label>
	<label>{m.labelPhone()}<input name="phone" type="tel" required bind:value={phone} /></label>
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={originV} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destinationV} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>

	<div id="alert-mode-btns" class="flex flex-wrap gap-2" role="group">
		{#each modes as mo}
			<button type="button" class="btn-mode" class:active={mode === mo.key} data-mode={mo.key} onclick={() => (mode = mo.key)}>{mo.label()}</button>
		{/each}
	</div>

	{#if mode !== 'anytime'}
		<div id="alert-time-fields" class="flex flex-col gap-2">
			<div class="search-datetime-row flex gap-2">
				{#if mode !== 'daily'}<label>{m.labelSearchDate()}<input name="alert_date" type="date" bind:value={alert_date} /></label>{/if}
				{#if mode !== 'day'}<label>{m.labelSearchTime()}<input name="alert_time" type="time" bind:value={alert_time} /></label>{/if}
			</div>
			{#if mode === 'time' || mode === 'daily'}
				<label>{m.labelFlex()}
					<select bind:value={flexibility}>
						<option value={0}>{m.flexExact()}</option>
						<option value={30}>{m.flex30()}</option>
						<option value={60}>{m.flex60()}</option>
					</select>
				</label>
			{/if}
		</div>
	{/if}

	<button type="submit" class="btn btn-primary">{m.notifEnable()}</button>
	{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
</form>
```
> Test 8 fills `alert_date`/`alert_time` then submits in the default `time` mode → posts `departure_at`. Tests 19/20 navigate from search "notify" with a `departureAt` prop, which pre-fills `alert_date`/`alert_time`.

- [ ] **Step 3: Implement AlertCard**

`frontend/src/lib/components/alerts/AlertCard.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime, formatDate, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Request } from '$lib/types';

	let { request, onseematches }: { request: Request; onseematches?: (o: string, d: string, dep: string) => void } = $props();
	let msg = $state('');
	let deleted = $state(false);

	const ZERO = '0001-01-01T00:00:00Z';
	const hasDate = $derived(request.Date !== ZERO && request.Date?.slice(0, 4) !== '0001');
	const hasTime = $derived(request.DepartureAt !== ZERO && request.DepartureAt?.slice(0, 4) !== '0001');
	const isDaily = $derived(hasTime && request.DepartureAt.slice(0, 10) === '1970-01-01');

	async function del() {
		try {
			await api.requests.del(request.ID, get(userPhone));
			msg = m.deleteOk();
			deleted = true;
		} catch {
			msg = m.deleteErr();
		}
	}
</script>

<div class="card rounded border p-3" id="card-{request.ID}" style:opacity={deleted ? 0.4 : 1}>
	<div class="card-route font-medium" translate="no">{m.alertCard({ origin: request.Origin, destination: request.Destination })}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		{#if !hasDate && !hasTime}
			<span class="tag tag-anytime">{m.alertAnytimeLabel()}</span>
		{:else if isDaily}
			<span class="tag tag-daily">{new Date(request.DepartureAt).toISOString().slice(11, 16)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{:else if hasDate && !hasTime}
			<span class="tag">{formatDate(request.Date)}</span>
		{:else}
			<span>{formatTime(request.DepartureAt)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{/if}
	</div>
	<div class="alert-actions flex gap-2">
		<button type="button" class="btn-see-matches" data-origin={request.Origin} data-dest={request.Destination}
			data-dept={hasTime ? request.DepartureAt : ''}
			onclick={() => onseematches?.(request.Origin, request.Destination, hasTime ? request.DepartureAt : '')}>{m.btnSeeMatches()}</button>
		<button type="button" class="btn btn-danger btn-delete" data-id={request.ID} data-phone={get(userPhone)} disabled={deleted} onclick={del}>{m.btnDelete()}</button>
	</div>
	<div class="delete-msg" id="msg-{request.ID}">{msg}</div>
</div>
```

- [ ] **Step 4: Run AlertForm test → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/alerts/AlertForm.test.ts`

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): AlertForm (4 modes) + AlertCard"
```

---

### Task 17: `RequestCard` — contact request (pending / accepted)

**Files:**
- Create: `frontend/src/lib/components/requests/RequestCard.svelte`
- Test: `frontend/src/lib/components/requests/RequestCard.test.ts`

**Test-hooks owned:** `.card`; for accepted, `.btn-contact-link` with `href="/interests/<id>"`.

- [ ] **Step 1: Failing test**

`frontend/src/lib/components/requests/RequestCard.test.ts`:
```typescript
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import RequestCard from './RequestCard.svelte';

const base = { id: 'int7', ride_id: 'r1', driver_name: 'Alice', origin: 'Saillans', destination: 'Crest', departure_at: '2030-06-15T08:00:00Z' };

describe('RequestCard', () => {
	it('pending shows the waiting label, no contact link', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'pending' } } });
		expect(container.querySelector('.btn-contact-link')).toBeNull();
	});
	it('accepted shows a contact link to /interests/<id>', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'accepted' } } });
		const a = container.querySelector('.btn-contact-link') as HTMLAnchorElement;
		expect(a.getAttribute('href')).toBe('/interests/int7');
	});
});
```

- [ ] **Step 2: Run → FAIL**, then implement.

`frontend/src/lib/components/requests/RequestCard.svelte`:
```svelte
<script lang="ts">
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { MyInterest } from '$lib/types';

	let { interest }: { interest: MyInterest } = $props();
	const accepted = $derived(interest.status === 'accepted' || interest.status === 'driver_shared');
</script>

<div class="card rounded border p-3" id="req-card-{interest.id}">
	<div class="card-route font-medium" translate="no">{interest.origin} → {interest.destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(interest.departure_at)}</span>
		<span>{interest.driver_name}</span>
		{#if accepted}
			<span class="tag tag-accepted">{m.reqStatusAccepted()}</span>
		{:else}
			<span class="tag">{m.reqStatusPending()}</span>
		{/if}
	</div>
	{#if accepted}
		<a class="btn-contact-link" href="/interests/{interest.id}">{m.contactRevealed()} →</a>
	{:else}
		<span class="interest-pending-label text-sm text-gray-600">{m.interestPending()}</span>
	{/if}
</div>
```

- [ ] **Step 3: Run → PASS**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/requests/RequestCard.test.ts`

- [ ] **Step 4: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): RequestCard"
```

---

### Task 18: Notifications — `BellButton`, `NotifModal`, `A2HSModal`, `A2HSBanner`, `PollToast`

**Files:**
- Create: `frontend/src/lib/a2hs.ts`
- Replace stubs: `frontend/src/lib/components/notifications/{BellButton,NotifModal,A2HSModal,A2HSBanner,PollToast}.svelte`
- Modify: `frontend/src/routes/+layout.svelte` (render `A2HSModal`; wire banner `onopen`)
- Test: `frontend/src/lib/components/notifications/NotifModal.test.ts`

- [ ] **Step 1: A2HS modal open-state**

`frontend/src/lib/a2hs.ts`:
```typescript
import { writable } from 'svelte/store';
export const a2hsModalOpen = writable(false);
export const openA2HS = () => a2hsModalOpen.set(true);
```

- [ ] **Step 2: BellButton**

`frontend/src/lib/components/notifications/BellButton.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { pushState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { openA2HS } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';

	let state = $derived($pushState);
	let isIos = $derived(state === 'ios-browser');
	let subscribed = $derived(state === 'subscribed');

	function click() {
		if (isIos) openA2HS();
		else openNotifModal(get(pushState));
	}
</script>

{#if isIos}
	<button type="button" class="bell-activate-label" onclick={openA2HS}>{m.a2hsHint()}</button>
{:else}
	<button type="button" class="btn-bell" class:bell-enabled={subscribed} aria-label="Notifications" data-notif-state={state} onclick={click}>🔔</button>
	{#if !subscribed}<button type="button" class="bell-activate-label" onclick={click}>{m.btnActivate()}</button>{/if}
{/if}
```

- [ ] **Step 3: NotifModal (state machine) — failing test first**

`frontend/src/lib/components/notifications/NotifModal.test.ts`:
```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import NotifModal from './NotifModal.svelte';
import { notifModalState } from '$lib/notifModal';

beforeEach(() => notifModalState.set(null));

describe('NotifModal', () => {
	it('is closed when state is null', () => {
		const { container } = render(NotifModal);
		expect(container.querySelector('.modal-overlay')).toBeNull();
	});
	it('default state shows enable + skip buttons', async () => {
		render(NotifModal);
		notifModalState.set('default');
		await Promise.resolve();
		expect(await screen.findByText(/Activer les notifications|Enable notifications/)).toBeInTheDocument();
	});
});
```

- [ ] **Step 4: Run → FAIL**, then implement NotifModal.

`frontend/src/lib/components/notifications/NotifModal.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { browser } from '$app/environment';
	import { notifModalState } from '$lib/notifModal';
	import { userPhone } from '$lib/stores';
	import { trySubscribePush, updateBellState } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';

	let state = $derived($notifModalState);
	let err = $state('');

	function close() { notifModalState.set(null); }

	async function enable() {
		err = '';
		let phone = get(userPhone);
		if (!phone && browser) phone = window.prompt(m.labelPhone()) ?? '';
		if (!phone) return;
		const perm = await Notification.requestPermission();
		if (perm === 'granted') {
			await trySubscribePush(phone);
			await updateBellState(phone);
			close();
		} else {
			notifModalState.set('denied');
		}
	}
</script>

{#if state}
	<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={close} role="presentation">
		<div class="modal w-full max-w-sm rounded bg-white p-5" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<h3 class="mb-2 text-lg font-semibold">{m.notifTitle()}</h3>
			{#if state === 'subscribed' || state === 'granted'}
				<p>{m.notifEnabled()}</p>
				<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
			{:else if state === 'denied'}
				<p>{m.notifDeniedTip()}</p>
				<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
			{:else}
				<p>{m.notifBody()}</p>
				<div class="mt-3 flex gap-2">
					<button type="button" id="btn-notif-modal-enable" class="btn btn-primary" onclick={enable}>{m.notifEnable()}</button>
					<button type="button" id="btn-notif-modal-skip" class="btn btn-secondary" onclick={close}>{m.notifSkip()}</button>
				</div>
				{#if err}<div class="mt-2 text-red-600">{err}</div>{/if}
			{/if}
		</div>
	</div>
{/if}
```

- [ ] **Step 5: A2HSModal**

`frontend/src/lib/components/notifications/A2HSModal.svelte`:
```svelte
<script lang="ts">
	import { a2hsModalOpen } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';
	function close() { a2hsModalOpen.set(false); }
</script>

{#if $a2hsModalOpen}
	<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={close} role="presentation">
		<div class="modal w-full max-w-sm rounded bg-white p-5" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
			<h3 class="mb-2 text-lg font-semibold">{m.a2hsTitle()}</h3>
			<p>{m.a2hsBody()}</p>
			<ol class="a2hs-steps mt-3 list-none space-y-1">
				<li class="a2hs-step">{m.a2hsStep1()}</li>
				<li class="a2hs-step">{m.a2hsStep2()}</li>
				<li class="a2hs-step">{m.a2hsStep3()}</li>
			</ol>
			<p class="a2hs-note mt-3 text-sm text-gray-500">{m.a2hsNote()}</p>
			<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
		</div>
	</div>
{/if}
```

- [ ] **Step 6: A2HSBanner**

`frontend/src/lib/components/notifications/A2HSBanner.svelte`:
```svelte
<script lang="ts">
	import { browser } from '$app/environment';
	import { isIOSBrowser } from '$lib/pwa';
	import { openA2HS } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';

	let dismissed = $state(browser ? localStorage.getItem('a2hs_dismissed') === '1' : true);
	let show = $derived(browser && isIOSBrowser() && !dismissed);

	function dismiss() {
		if (browser) localStorage.setItem('a2hs_dismissed', '1');
		dismissed = true;
	}
</script>

{#if show}
	<div id="a2hs-banner" class="a2hs-banner mx-auto flex max-w-xl items-center gap-2 rounded bg-blue-50 p-2">
		<span>🔔 {m.a2hsTitle()}</span>
		<button type="button" id="a2hs-banner-open" class="a2hs-banner-cta underline" onclick={openA2HS}>{m.a2hsHint()}</button>
		<button type="button" id="a2hs-banner-dismiss" class="a2hs-banner-dismiss ml-auto" aria-label="Dismiss" onclick={dismiss}>✕</button>
	</div>
{/if}
```

- [ ] **Step 7: PollToast host**

`frontend/src/lib/components/notifications/PollToast.svelte`:
```svelte
<script lang="ts">
	import { goto } from '$app/navigation';
	import { pollToasts } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';
	import type { NotificationItem } from '$lib/types';

	function dismiss(n: NotificationItem) { pollToasts.update((t) => t.filter((x) => x !== n)); }
	function view(n: NotificationItem) { dismiss(n); goto('/my-searches'); }
</script>

<div class="poll-toast-host fixed bottom-3 left-1/2 z-50 flex -translate-x-1/2 flex-col gap-2">
	{#each $pollToasts as n}
		<div class="poll-toast flex items-center gap-2 rounded bg-gray-900 p-3 text-white shadow">
			<span class="poll-toast-body">🚗 {n.driver_name} · {n.origin} → {n.destination}</span>
			<button type="button" class="poll-toast-view underline" onclick={() => view(n)}>{m.pollToastView()}</button>
			<button type="button" class="poll-toast-close" aria-label="Dismiss" onclick={() => dismiss(n)}>✕</button>
		</div>
	{/each}
</div>
```

- [ ] **Step 8: Render A2HSModal in the layout, wire banner open**

In `+layout.svelte`, import and render `<A2HSModal />` (add `import A2HSModal from '$lib/components/notifications/A2HSModal.svelte';` and place `<A2HSModal />` near the other global components). Replace the banner line with `<A2HSBanner />` (it manages its own open via `openA2HS`). Also open the standalone notif prompt on mount: in the `onMount` `isStandalone()` branch, call `openNotifModal('default')` (import `openNotifModal` from `$lib/notifModal`).

- [ ] **Step 9: Run NotifModal test → PASS, build**

Run: `cd frontend && npm run test:unit -- --run src/lib/components/notifications/NotifModal.test.ts && npm run build`
Expected: test passes; build succeeds.

- [ ] **Step 10: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): notifications (bell, notif modal, A2HS modal+banner, poll toast)"
```

---

# Phase 3 — Routes (wire components)

Outcome of Phase 3: every route renders. Route integration is verified end-to-end by the Playwright suite in Task 30; per-route tasks build + smoke-check. Each task lists its data loads and the Test-Hook Contract entries the page owns.

### Task 19: Shared `loadAcceptedContacts` + `/` home

**Files:**
- Create: `frontend/src/lib/contacts.ts`
- Create: `frontend/src/routes/+page.svelte`
- Test: `frontend/src/routes/home.test.ts`

**Test-hooks owned:** `<h1>` (contains `siteName`); `button.btn-primary` ("Je conduis"); `button.btn-secondary`; three `.btn-ghost-inline`.

- [ ] **Step 1: Accepted-contacts helper**

`frontend/src/lib/contacts.ts`:
```typescript
import { browser } from '$app/environment';
import { api } from './api';
import type { MyInterest } from './types';

/**
 * For the given phone, fetch the searcher's own interests, sync accepted/driver_shared
 * ones into localStorage (interest_<rideId>), and return a map rideId -> contact phone.
 */
export async function loadAcceptedContacts(phone: string): Promise<Map<string, string>> {
	const map = new Map<string, string>();
	if (!phone) return map;
	let mine: MyInterest[] = [];
	try {
		mine = await api.interests.listMine(phone);
	} catch {
		return map;
	}
	for (const it of mine) {
		if (browser) localStorage.setItem(`interest_${it.ride_id}`, it.id);
		if (it.status === 'accepted' || it.status === 'driver_shared') {
			try {
				const c = await api.interests.getContact(it.id, phone);
				map.set(it.ride_id, c.phone);
			} catch {
				/* ignore */
			}
		}
	}
	return map;
}
```

- [ ] **Step 2: Failing home test**

`frontend/src/routes/home.test.ts`:
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import Home from './+page.svelte';

afterEach(() => vi.unstubAllGlobals());

describe('home', () => {
	it('renders hero CTAs and ghost nav', () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Home);
		expect(container.querySelector('button.btn-primary')?.textContent).toMatch(/Je conduis|driving/);
		expect(container.querySelector('button.btn-secondary')).toBeTruthy();
		expect(container.querySelectorAll('.btn-ghost-inline').length).toBe(3);
	});
});
```

- [ ] **Step 3: Run → FAIL**, then implement home.

`frontend/src/routes/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { config } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import RideCard from '$lib/components/rides/RideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Stats } from '$lib/types';

	let rides = $state<PublicRide[]>([]);
	let contacts = $state<Map<string, string>>(new Map());
	let stats = $state<Stats | null>(null);
	let pendingBadge = $state(0);

	onMount(async () => {
		const phone = get(userPhone);
		contacts = await loadAcceptedContacts(phone);
		try { rides = (await api.rides.list()) as PublicRide[]; } catch { rides = []; }
		try { stats = await api.stats.get(); } catch { stats = null; }
		// pending-interest badge: count pending interests across my own rides
		if (phone) {
			try {
				const mine = (await api.rides.list({}, phone)) as { ID: string }[];
				let n = 0;
				for (const r of mine) {
					const ints = await api.rides.listInterests(r.ID, phone);
					n += ints.filter((i) => i.status === 'pending').length;
				}
				pendingBadge = n;
			} catch { pendingBadge = 0; }
		}
	});
</script>

<div class="hero text-center">
	<h1 class="text-2xl font-bold">{$config.siteName}</h1>
	<p class="tagline text-gray-600">{m.tagline()}</p>
	<div class="mt-4 flex flex-col gap-2">
		<button type="button" class="btn btn-primary" onclick={() => goto('/post-ride')}>{m.btnDriver()}</button>
		<button type="button" class="btn btn-secondary" onclick={() => goto('/search')}>{m.btnSearcher()}</button>
	</div>
	<div class="ghost-row mt-3 flex items-center justify-center gap-2 text-sm">
		<button type="button" class="btn-ghost-inline" onclick={() => goto('/me')}>{m.btnMe()}</button>
		<span class="ghost-sep">·</span>
		<button type="button" class="btn-ghost-inline relative" onclick={() => goto('/my-rides')}>
			{m.btnMyRides()}{#if pendingBadge > 0}<span class="interest-badge ml-1 rounded-full bg-red-500 px-1 text-white">{pendingBadge}</span>{/if}
		</button>
		<span class="ghost-sep">·</span>
		<button type="button" class="btn-ghost-inline" onclick={() => goto('/my-searches')}>{m.btnMySearches()}</button>
	</div>
</div>

<section id="home-feed" class="mt-5">
	<h2 class="home-feed-title mb-2 font-semibold">{m.homeFeedTitle()}</h2>
	{#if rides.length === 0}
		<p class="home-feed-empty text-gray-500">{m.noActiveRides()}</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each rides as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} />{/each}
		</div>
	{/if}
</section>

<section id="home-stats" class="mt-5">
	{#if stats && stats.total_confirmed > 0}
		<div class="stats-widget rounded border p-3">
			<div class="stats-widget-title font-semibold">{m.statsAllTime({ n: stats.total_confirmed })}</div>
			{#each stats.top_routes as rt}
				<button type="button" class="stats-row stats-row-btn block w-full text-left" data-origin={rt.Origin} data-dest={rt.Destination}
					onclick={() => goto(`/search?origin=${encodeURIComponent(rt.Origin)}&destination=${encodeURIComponent(rt.Destination)}`)}>
					{rt.Origin} → {rt.Destination} <span class="stats-count">{m.statsRouteCount({ n: rt.Count })}</span>
				</button>
			{/each}
			<a class="btn-all-stats underline" href="/stats">{m.btnAllStats()}</a>
		</div>
	{/if}
</section>
```

- [ ] **Step 4: Run → PASS, build**

Run: `cd frontend && npm run test:unit -- --run src/routes/home.test.ts && npm run build`

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): home route + accepted-contacts helper"
```

---

### Task 20: `/post-ride`

**Files:**
- Create: `frontend/src/routes/post-ride/+page.svelte`

- [ ] **Step 1: Implement**

`frontend/src/routes/post-ride/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import RideForm from '$lib/components/rides/RideForm.svelte';
	import { openNotifModal } from '$lib/notifModal';
	import { pushState } from '$lib/pwa';
	import { get } from 'svelte/store';
	import { m } from '$lib/paraglide/messages';

	let destinations = $state<string[]>([]);
	onMount(async () => { try { destinations = await api.destinations.list(); } catch { destinations = []; } });

	function posted() {
		if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		goto('/my-rides');
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.postRideTitle()}</h2>
<RideForm {destinations} onposted={posted} />
```
> Test 3 navigates here (`button.btn-primary` on home), fills the form, submits (expects `POST /api/rides` 201), and lands on `/my-rides` showing the `.card-route` "Saillans → Crest". The notification prompt is a modal overlay (does not block navigation) — and tests that need it suppressed call `grantPermissions(['notifications'])`, so `pushState` becomes `subscribed`/`granted` and the modal is skipped or shows the "enabled" branch.

- [ ] **Step 2: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): post-ride route"
```

---

### Task 21: `/post-request` (AlertForm route) — D4

**Files:**
- Create: `frontend/src/routes/post-request/+page.svelte`

- [ ] **Step 1: Implement**

`frontend/src/routes/post-request/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import AlertForm from '$lib/components/alerts/AlertForm.svelte';
	import { openNotifModal } from '$lib/notifModal';
	import { pushState } from '$lib/pwa';
	import { get } from 'svelte/store';
	import { m } from '$lib/paraglide/messages';

	let destinations = $state<string[]>([]);
	const origin = $derived(page.url.searchParams.get('origin') ?? '');
	const destination = $derived(page.url.searchParams.get('destination') ?? '');
	const departureAt = $derived(page.url.searchParams.get('departure_at') ?? '');

	onMount(async () => { try { destinations = await api.destinations.list(); } catch { destinations = []; } });

	function posted() {
		if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		goto('/');
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.notifRouteTitle()}</h2>
<p class="section-hint mb-3 text-sm text-gray-600">{m.notifRouteBody()}</p>
<AlertForm {origin} {destination} {departureAt} {destinations} onposted={posted} />
```
> Test 8 navigates to `/post-request?origin=Saillans&destination=Crest`, fills the alert form across all 4 modes, submits (expects `POST /api/requests` 201). The `#notify-form` and `.btn-mode` hooks come from `AlertForm`.

- [ ] **Step 2: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): post-request (AlertForm) route"
```

---

### Task 22: `/search` — forward + return columns, notify buttons, URL round-trip

**Files:**
- Create: `frontend/src/routes/search/+page.svelte`
- Test: `frontend/src/routes/search.test.ts`

**Test-hooks owned:** `#search-form`; input `name`s `origin`,`destination`,`search_date`,`search_time`; `.results-col-header` (2); `.col-notify`; `.col-empty`. URL params round-trip on reload.

- [ ] **Step 1: Failing test (two result columns render)**

`frontend/src/routes/search.test.ts`:
```typescript
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import Search from './+page.svelte';

afterEach(() => vi.unstubAllGlobals());

vi.mock('$app/state', () => ({
	page: { url: new URL('http://localhost/search?origin=Saillans&destination=Crest') }
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

describe('search', () => {
	it('renders two result columns after an auto-submitted query', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Search);
		await vi.waitFor(() => expect(container.querySelectorAll('.results-col-header').length).toBe(2));
	});
});
```

- [ ] **Step 2: Run → FAIL**, then implement search.

`frontend/src/routes/search/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { lastOrigin, lastDestination, userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import RideCard from '$lib/components/rides/RideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, RideSearchParams } from '$lib/types';

	let destinations = $state<string[]>([]);
	let origin = $state('');
	let destination = $state('');
	let search_date = $state('');
	let search_time = $state('');
	let submitted = $state(false);
	let fwd = $state<PublicRide[]>([]);
	let rev = $state<PublicRide[]>([]);
	let contacts = $state<Map<string, string>>(new Map());

	function fromUrl() {
		const sp = page.url.searchParams;
		origin = sp.get('origin') ?? '';
		destination = sp.get('destination') ?? '';
		search_date = sp.get('search_date') ?? '';
		search_time = sp.get('search_time') ?? '';
		const dep = sp.get('departure_at');
		if (dep) { // split a UTC instant into local date + time for the inputs
			const d = new Date(dep);
			const p = (n: number) => String(n).padStart(2, '0');
			search_date = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
			search_time = `${p(d.getHours())}:${p(d.getMinutes())}`;
		}
	}

	async function run() {
		if (!origin || !destination) return;
		submitted = true;
		lastOrigin.set(origin);
		lastDestination.set(destination);

		const params: RideSearchParams = { origin, destination };
		const url = new URLSearchParams({ origin, destination });
		if (search_date && search_time) {
			const iso = new Date(`${search_date}T${search_time}`).toISOString();
			params.departure_at = iso;
			url.set('departure_at', iso);
		} else if (search_date) {
			params.search_date = search_date; url.set('search_date', search_date);
		} else if (search_time) {
			params.search_time = search_time; url.set('search_time', search_time);
		}
		goto(`/search?${url.toString()}`, { replaceState: true, keepFocus: true, noScroll: true });

		const revParams: RideSearchParams = { ...params, origin: destination, destination: origin };
		const [a, b] = await Promise.all([
			api.rides.list(params).catch(() => []),
			api.rides.list(revParams).catch(() => [])
		]);
		fwd = a as PublicRide[];
		rev = b as PublicRide[];
		contacts = await loadAcceptedContacts(get(userPhone));
	}

	function submit(e: SubmitEvent) { e.preventDefault(); run(); }
	function notify(o: string, d: string) {
		const u = new URLSearchParams({ origin: o, destination: d });
		if (search_date && search_time) u.set('departure_at', new Date(`${search_date}T${search_time}`).toISOString());
		goto(`/post-request?${u.toString()}`);
	}

	onMount(async () => {
		try { destinations = await api.destinations.list(); } catch { destinations = []; }
		fromUrl();
		if (!origin && !destination) { origin = get(lastOrigin); destination = get(lastDestination); }
		if (origin && destination) run();
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.findTitle()}</h2>
<form id="search-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={origin} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destination} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<div class="search-datetime-row flex gap-2">
		<label>{m.labelSearchDate()}<input name="search_date" type="date" bind:value={search_date} /></label>
		<label>{m.labelSearchTime()}<input name="search_time" type="time" bind:value={search_time} /></label>
	</div>
	<button type="submit" class="btn btn-primary">{m.btnSearch()}</button>
</form>

{#if submitted}
	<div id="results" class="results-grid mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
		{#each [{ list: fwd, o: origin, d: destination }, { list: rev, o: destination, d: origin }] as col}
			<div class="results-col">
				<div class="results-col-header font-semibold" translate="no">{col.o} → {col.d}</div>
				{#if col.list.length === 0}
					<div class="col-empty">
						<p>{m.noRidesCol()}</p>
						<button type="button" class="btn-notify-route col-notify underline" data-from={col.o} data-to={col.d} onclick={() => notify(col.o, col.d)}>{m.btnNotifyRoute()}</button>
					</div>
				{:else}
					<div class="flex flex-col gap-2">
						{#each col.list as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} />{/each}
						<button type="button" class="btn-notify-route col-notify underline" data-from={col.o} data-to={col.d} onclick={() => notify(col.o, col.d)}>{m.btnNotifyRoute()}</button>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
```
> Notes for the e2e suite: two `.results-col-header` always render once submitted (tests 15,16,17,18). An empty column shows `.col-empty` + `.col-notify`; a non-empty column shows cards + a trailing `.col-notify` (test 17). Unknown route → both columns empty → exactly 2 `.col-notify` (test 18). `search_time`-only submit issues `GET /api/rides?...&search_time=` (test 26). URL round-trips `origin/destination/search_date/search_time/departure_at` for reload (tests 13,14,23,24,25).

- [ ] **Step 3: Run → PASS, build**

Run: `cd frontend && npm run test:unit -- --run src/routes/search.test.ts && npm run build`

- [ ] **Step 4: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): search route (forward/return columns, notify, URL round-trip)"
```

---

### Task 23: `MyRideCard` + `/my-rides`

**Files:**
- Create: `frontend/src/lib/components/rides/MyRideCard.svelte`
- Create: `frontend/src/routes/my-rides/+page.svelte`

**Test-hooks owned:** `#my-rides-form`, `#my-rides-form button[type=submit]`; `.card`, `.card-route`; seekers via `SeekerRow` (`.seeker-row`, `.btn-ping-searcher`); `.btn-delete`. (Test 7: `#app` innerText contains the searcher name; test 29: ping flow.)

- [ ] **Step 1: Implement MyRideCard (seekers + interests + feedback + delete)**

`frontend/src/lib/components/rides/MyRideCard.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { formatTime, flexLabel } from '$lib/utils';
	import SeekerRow from './SeekerRow.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Ride, PublicRequest, InterestListItem } from '$lib/types';

	let { ride, phone }: { ride: Ride; phone: string } = $props();
	let seekers = $state<PublicRequest[]>([]);
	let interests = $state<InterestListItem[]>([]);
	let deleted = $state(false);
	let delMsg = $state('');
	let fbDone = $state(false);

	const isPast = $derived(new Date(ride.DepartureAt).getTime() < Date.now());
	const showFeedback = $derived(isPast && !ride.FeedbackGiven && !fbDone);

	onMount(async () => {
		try { seekers = await api.rides.listMatchingRequests(ride.ID, phone); } catch { seekers = []; }
		try { interests = await api.rides.listInterests(ride.ID, phone); } catch { interests = []; }
	});

	async function accept(it: InterestListItem) {
		try {
			const res = await api.interests.accept(it.id, phone);
			interests = interests.map((x) => x.id === it.id ? { ...x, status: 'accepted', searcher_phone: res.searcher_phone } : x);
		} catch { /* surfaced inline below if needed */ }
	}
	async function feedback(taken: boolean) {
		try { await api.rides.feedback(ride.ID, phone, taken); fbDone = true; } catch { /* ignore */ }
	}
	async function del() {
		try { await api.rides.del(ride.ID, phone); deleted = true; delMsg = m.deleteOk(); }
		catch { delMsg = m.deleteErr(); }
	}
	const pendingCount = $derived(interests.filter((i) => i.status === 'pending').length);
</script>

<div class="card rounded border p-3" id="card-{ride.ID}" style:opacity={deleted ? 0.4 : 1}>
	<div class="card-route font-medium" translate="no">{ride.Origin} → {ride.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(ride.DepartureAt)}</span>
		<span class="tag">{flexLabel(ride.Flexibility)}</span>
	</div>

	<div class="seekers-section" id="seekers-{ride.ID}">
		{#if seekers.length > 0}
			<div class="seekers-title mt-2 text-sm font-medium">{m.seekersTitle()}</div>
			{#each seekers as s}<SeekerRow request={s} rideId={ride.ID} driverPhone={phone} />{/each}
		{:else}
			<div class="seekers-empty text-sm text-gray-500">{m.noSeekers()}</div>
		{/if}
	</div>

	<div class="interests-section" id="interests-{ride.ID}">
		{#if pendingCount > 0}<div class="interests-title mt-2 text-sm font-medium">{m.pendingInterests({ count: pendingCount })}</div>{/if}
		{#each interests as it}
			<div class="interest-row" id="irow-{it.id}">
				{#if it.status === 'pending'}
					<span class="interest-pending-info">{it.searcher_name ?? ''}</span>
					<button type="button" class="btn-accept-interest" data-id={it.id} data-phone={phone} onclick={() => accept(it)}>{m.btnAccept()}</button>
				{:else if it.status === 'driver_shared'}
					<span class="interest-accepted">{m.notifSentShort()}</span>
				{:else}
					<span class="interest-accepted">{m.contactRevealed()}{#if it.searcher_phone}: <a href="tel:{it.searcher_phone}">{it.searcher_phone}</a>{/if}</span>
				{/if}
			</div>
		{/each}
	</div>

	{#if showFeedback}
		<div class="feedback-prompt mt-2" id="fb-{ride.ID}">
			<div class="feedback-question text-sm">{m.feedbackTitle()}</div>
			<div class="feedback-btns flex gap-2">
				<button type="button" class="btn-fb-yes" data-id={ride.ID} data-phone={phone} onclick={() => feedback(true)}>{m.feedbackYes()}</button>
				<button type="button" class="btn-fb-no" data-id={ride.ID} data-phone={phone} onclick={() => feedback(false)}>{m.feedbackNo()}</button>
			</div>
		</div>
	{:else if fbDone}
		<div class="feedback-thanks text-sm text-green-600">{m.feedbackThanks()}</div>
	{/if}

	<button type="button" class="btn btn-danger btn-delete" data-id={ride.ID} data-phone={phone} disabled={deleted} onclick={del}>{m.btnDelete()}</button>
	<div class="delete-msg" id="msg-{ride.ID}">{delMsg}</div>
</div>
```

- [ ] **Step 2: Implement the my-rides route**

`frontend/src/routes/my-rides/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import MyRideCard from '$lib/components/rides/MyRideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Ride } from '$lib/types';

	let phone = $state(get(userPhone));
	let rides = $state<Ride[]>([]);
	let loaded = $state(false);

	async function load(e?: SubmitEvent) {
		e?.preventDefault();
		userPhone.set(phone);
		try { rides = (await api.rides.list({}, phone)) as Ride[]; } catch { rides = []; }
		loaded = true;
	}
	onMount(async () => { if (phone) { await tick(); load(); } });
</script>

<h2 class="mb-3 text-xl font-semibold">{m.myRidesTitle()}</h2>
<form id="my-rides-form" onsubmit={load} class="mb-4 flex items-end gap-2">
	<label class="grow">{m.labelPhoneCheck()}<input name="phone" type="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnShowRides()}</button>
</form>

<div id="my-rides-list" class="flex flex-col gap-3">
	{#if loaded && rides.length === 0}
		<p class="empty text-gray-500">{m.noMyRides()}</p>
	{:else}
		{#each rides as r (r.ID)}<MyRideCard ride={r} {phone} />{/each}
	{/if}
</div>
```

- [ ] **Step 3: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): my-rides route + MyRideCard (seekers, interests, feedback, delete)"
```

---

### Task 24: `/my-searches` + `/my-alerts` & `/my-requests` redirects

**Files:**
- Create: `frontend/src/routes/my-searches/+page.svelte`
- Create: `frontend/src/routes/my-alerts/+page.ts`
- Create: `frontend/src/routes/my-requests/+page.ts`

**Test-hooks owned:** `#my-searches-form`, `#my-searches-form button[type=submit]`; `#my-alerts-list` (children > 0); `.card`, `.btn-delete`, `.delete-msg` (via `AlertCard`).

- [ ] **Step 1: Redirects**

`frontend/src/routes/my-alerts/+page.ts`:
```typescript
import { redirect } from '@sveltejs/kit';
export const load = () => { throw redirect(307, '/my-searches'); };
```
`frontend/src/routes/my-requests/+page.ts`:
```typescript
import { redirect } from '@sveltejs/kit';
export const load = () => { throw redirect(307, '/my-searches'); };
```

- [ ] **Step 2: my-searches page**

`frontend/src/routes/my-searches/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import AlertCard from '$lib/components/alerts/AlertCard.svelte';
	import RequestCard from '$lib/components/requests/RequestCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Request, MyInterest } from '$lib/types';

	let phone = $state(get(userPhone));
	let alerts = $state<Request[]>([]);
	let requests = $state<MyInterest[]>([]);
	let loaded = $state(false);

	async function load(e?: SubmitEvent) {
		e?.preventDefault();
		userPhone.set(phone);
		const [a, r] = await Promise.all([
			api.requests.list(phone).catch(() => []),
			api.interests.listMine(phone).catch(() => [])
		]);
		alerts = a; requests = r; loaded = true;
	}
	function seeMatches(o: string, d: string, dep: string) {
		const u = new URLSearchParams({ origin: o, destination: d });
		if (dep) u.set('departure_at', dep);
		goto(`/search?${u.toString()}`);
	}
	onMount(async () => { if (phone) { await tick(); load(); } });
</script>

<h2 class="mb-3 text-xl font-semibold">{m.mySearchesTitle()}</h2>
<form id="my-searches-form" onsubmit={load} class="mb-4 flex items-end gap-2">
	<label class="grow">{m.labelPhoneCheck()}<input name="phone" type="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnShowSearches()}</button>
</form>

<div id="my-searches-content">
	<section>
		<div class="section-label font-semibold">{m.myAlertsTitle()}</div>
		<div id="my-alerts-list" class="flex flex-col gap-2">
			{#if loaded && alerts.length === 0}
				<p class="empty text-gray-500">{m.noMyAlerts()}</p>
			{:else}
				{#each alerts as a (a.ID)}<AlertCard request={a} onseematches={seeMatches} />{/each}
			{/if}
		</div>
	</section>
	<section class="mt-5">
		<div class="section-label font-semibold">{m.myRequestsTitle()}</div>
		<div id="my-requests-list" class="flex flex-col gap-2">
			{#if loaded && requests.length === 0}
				<p class="empty text-gray-500">{m.noMyRequests()}</p>
			{:else}
				{#each requests as r (r.id)}<RequestCard interest={r} />{/each}
			{/if}
		</div>
	</section>
</div>
```
> Test 9 submits `#my-searches-form`, waits for `GET /api/requests` 200, asserts `#my-alerts-list` has children, then clicks the first `.btn-delete` and asserts `.delete-msg` is non-empty. `AlertCard` handles its own delete and writes `.delete-msg`.

- [ ] **Step 3: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): my-searches route + my-alerts/my-requests redirects"
```

---

### Task 25: `/me`

**Files:**
- Create: `frontend/src/routes/me/+page.svelte`
- Test: `frontend/src/routes/me.test.ts`

**Test-hooks owned:** `#me-form`; input `name`s `name`,`phone`; submit; `#me-saved` (toggles inline `display:none`); `<h2>` matching `/Mon profil|My profile/`.

- [ ] **Step 1: Failing test**

`frontend/src/routes/me.test.ts`:
```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Me from './+page.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => { localStorage.clear(); userName.set(''); userPhone.set(''); });

describe('me', () => {
	it('saves to localStorage and reveals #me-saved', async () => {
		const { container } = render(Me);
		await fireEvent.input(container.querySelector('input[name=name]')!, { target: { value: 'Marie' } });
		await fireEvent.input(container.querySelector('input[name=phone]')!, { target: { value: '0644000001' } });
		await fireEvent.submit(container.querySelector('#me-form')!);
		expect(localStorage.getItem('user_name')).toBe(JSON.stringify('Marie'));
		const saved = container.querySelector('#me-saved') as HTMLElement;
		expect(saved.getAttribute('style') ?? '').not.toContain('none');
	});

	it('pre-fills from an existing profile', () => {
		userName.set('Jean'); userPhone.set('0655000002');
		const { container } = render(Me);
		expect((container.querySelector('input[name=name]') as HTMLInputElement).value).toBe('Jean');
		expect((container.querySelector('input[name=phone]') as HTMLInputElement).value).toBe('0655000002');
	});
});
```

- [ ] **Step 2: Run → FAIL**, then implement.

`frontend/src/routes/me/+page.svelte`:
```svelte
<script lang="ts">
	import { get } from 'svelte/store';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';

	let name = $state(get(userName));
	let phone = $state(get(userPhone));
	let saved = $state(false);

	function submit(e: SubmitEvent) {
		e.preventDefault();
		userName.set(name);
		userPhone.set(normalizePhone(phone));
		saved = true;
		setTimeout(() => (saved = false), 2000);
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.meTitle()}</h2>
<form id="me-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={name} /></label>
	<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnSave()}</button>
	<div id="me-saved" class="section-hint text-green-600" style:display={saved ? 'block' : 'none'}>{m.meSaved()}</div>
</form>
<p class="section-hint text-sm text-gray-500">{m.meHint()}</p>
```
> `style:display={saved ? 'block' : 'none'}` produces inline `style="display: block"` when saved (no "none"), satisfying test 32's `#me-saved:not([style*="none"])`, and `display: none` when not (test setup baseline).

- [ ] **Step 3: Run → PASS, build, commit**

Run: `cd frontend && npm run test:unit -- --run src/routes/me.test.ts && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): me route"
```

---

### Task 26: `/stats`

**Files:**
- Create: `frontend/src/routes/stats/+page.svelte`

**Test-hooks owned:** `<h2>` = `statsPageTitle` ("Statistiques"). (Test 2 reaches stats via the footer link, then back.)

- [ ] **Step 1: Implement**

`frontend/src/routes/stats/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { m } from '$lib/paraglide/messages';
	import type { Stats, ActivityCounts } from '$lib/types';

	let stats = $state<Stats | null>(null);
	let err = $state(false);
	onMount(async () => { try { stats = await api.stats.get(); } catch { err = true; } });

	const tables = $derived(stats ? [
		{ title: m.statsSearches(), c: stats.searches },
		{ title: m.statsRidesPosted(), c: stats.rides_posted }
	] : []);
	function rows(c: ActivityCounts) {
		return [
			{ label: m.statsThisMonth(), n: c.this_month },
			{ label: m.statsThisYear(), n: c.this_year },
			{ label: m.statsAllTime2(), n: c.all_time }
		];
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.statsPageTitle()}</h2>
<div id="stats-content">
	{#if err}
		<p class="error text-red-600">⚠</p>
	{:else if stats}
		{#if stats.total_confirmed > 0}<div class="stats-total font-semibold">{m.statsAllTime({ n: stats.total_confirmed })}</div>{/if}
		<div class="stats-week-title mt-2 font-medium">{m.statsTitle()}</div>
		{#if stats.top_routes.length > 0}
			{#each stats.top_routes as rt}
				<div class="stats-row flex justify-between"><span translate="no">{rt.Origin} → {rt.Destination}</span><span class="stats-count">{m.statsRouteCount({ n: rt.Count })}</span></div>
			{/each}
		{:else}
			<p class="section-hint text-gray-500">{m.statsEmpty()}</p>
		{/if}
		<div class="activity-stats mt-4 grid grid-cols-2 gap-4">
			{#each tables as t}
				<div class="activity-stat">
					<div class="activity-stat-title font-medium">{t.title}</div>
					<div class="activity-stat-rows">
						{#each rows(t.c) as r}<div class="activity-row flex justify-between"><span>{r.label}</span><span>{r.n}</span></div>{/each}
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<p>…</p>
	{/if}
</div>
```

- [ ] **Step 2: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): stats route"
```

---

### Task 27: `/interests/[id]` — contact reveal

**Files:**
- Create: `frontend/src/routes/interests/[id]/+page.svelte`

- [ ] **Step 1: Implement**

`frontend/src/routes/interests/[id]/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { ContactInfo } from '$lib/types';

	let contact = $state<ContactInfo | null>(null);
	let err = $state('');
	const id = $derived(page.params.id);

	onMount(async () => {
		try { contact = await api.interests.getContact(id, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.contactRevealed()}</h2>
<div id="contact-result">
	{#if err}
		<p class="error text-red-600">{err}</p>
	{:else if contact}
		<div class="card contact-card rounded border p-3">
			<div class="card-route font-medium" translate="no">{contact.origin} → {contact.destination}</div>
			<div class="card-meta text-sm text-gray-600">{formatTime(contact.departure_at)}</div>
			<div class="detail-table mt-2">
				<div>{contact.role === 'driver' ? m.labelDriver() : m.labelSearcher()}: {contact.name}</div>
				<div>{m.theirNumber()} <a href="tel:{contact.phone}">{contact.phone}</a></div>
			</div>
			<a class="btn btn-primary" href="tel:{contact.phone}">{m.btnCallNow()}</a>
			<button type="button" class="btn btn-secondary" id="btn-search-route" data-origin={contact.origin} data-dest={contact.destination}
				onclick={() => goto(`/search?origin=${encodeURIComponent(contact.origin)}&destination=${encodeURIComponent(contact.destination)}`)}>{m.btnSearchRoute()}</button>
		</div>
	{:else}
		<p>…</p>
	{/if}
</div>
```

- [ ] **Step 2: Build + commit**

Run: `cd frontend && npm run build`
```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): interests/[id] contact route"
```

---

### Task 28: `/rides/[id]` + `/requests/[id]` — push deep links

**Files:**
- Create: `frontend/src/routes/rides/[id]/+page.svelte`
- Create: `frontend/src/routes/requests/[id]/+page.svelte`

- [ ] **Step 1: Ride detail**

`frontend/src/routes/rides/[id]/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Ride } from '$lib/types';

	let ride = $state<Ride | null>(null);
	let err = $state('');
	onMount(async () => {
		try { ride = await api.rides.get(page.params.id); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.detailRideTitle()}</h2>
{#if err}
	<p class="error text-red-600">{err}</p>
{:else if ride}
	<div class="card detail-card rounded border p-3">
		<div class="card-route font-medium" translate="no">{ride.Origin} → {ride.Destination}</div>
		<div class="card-meta flex gap-2 text-sm text-gray-600"><span>{formatTime(ride.DepartureAt)}</span><span class="tag">{flexLabel(ride.Flexibility)}</span></div>
		<div class="detail-table mt-2">
			<div>{m.labelDriver()}: {ride.DriverName}</div>
			<div>{m.labelContact()}: <a href="tel:{ride.Phone}">{ride.Phone}</a></div>
		</div>
	</div>
{:else}
	<p>…</p>
{/if}
```

- [ ] **Step 2: Request detail** (`/requests/[id]`) — same shape, using `api.requests.get(id, phone)` with the profile phone, `detailReqTitle`, `labelSearcher`, `SearcherName`:

`frontend/src/routes/requests/[id]/+page.svelte`:
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Request } from '$lib/types';

	let req = $state<Request | null>(null);
	let err = $state('');
	onMount(async () => {
		try { req = await api.requests.get(page.params.id, get(userPhone)); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.detailReqTitle()}</h2>
{#if err}
	<p class="error text-red-600">{err}</p>
{:else if req}
	<div class="card detail-card rounded border p-3">
		<div class="card-route font-medium" translate="no">{req.Origin} → {req.Destination}</div>
		<div class="card-meta flex gap-2 text-sm text-gray-600"><span>{formatTime(req.DepartureAt)}</span><span class="tag">{flexLabel(req.Flexibility)}</span></div>
		<div class="detail-table mt-2">
			<div>{m.labelSearcher()}: {req.SearcherName}</div>
			<div>{m.labelContact()}: <a href="tel:{req.Phone}">{req.Phone}</a></div>
		</div>
	</div>
{:else}
	<p>…</p>
{/if}
```

- [ ] **Step 3: Build the whole app + run the full unit suite**

Run: `cd frontend && npm run build && npm run test:unit -- --run`
Expected: build succeeds; all unit tests pass.

- [ ] **Step 4: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend && git commit -m "feat(frontend): rides/[id] + requests/[id] push deep-link routes"
```

---

# Phase 4 — PWA assets, e2e migration, cleanup

Outcome of Phase 4: PWA assets ship from the SvelteKit build, all 32 Playwright tests pass against the Go-served build, and the legacy frontend is deleted.

### Task 29: PWA static assets + service-worker registration

**Files:**
- Create: `frontend/static/{sw.js,manifest.json,logo.svg,icon-192.png,icon-512.png,icon-maskable-192.png,icon-maskable-512.png,apple-touch-icon.png}`
- Modify: `.gitignore` (whitelist `frontend/static/*.png`)
- Modify: `frontend/src/routes/+layout.svelte` (register `/sw.js`)

- [ ] **Step 1: Copy the PWA assets verbatim into `frontend/static/`**

Run from repo root:
```bash
cp web/manifest.json web/logo.svg frontend/static/
cp web/icon-192.png web/icon-512.png web/icon-maskable-192.png web/icon-maskable-512.png web/apple-touch-icon.png frontend/static/
cp web/js/sw.js frontend/static/sw.js
```
`frontend/static/sw.js` content (unchanged push-only worker — verify it matches):
```javascript
'use strict';

self.addEventListener('push', (event) => {
	let data = {};
	try { data = event.data.json(); } catch {}
	const title = data.title || 'Go-Stop';
	const options = {
		body: data.body || '',
		icon: '/icon-192.png',
		badge: '/icon-192.png',
		data: { url: data.url || '/' }
	};
	event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const url = event.notification.data?.url || '/';
	event.waitUntil(
		clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
			for (const client of clientList) {
				if (client.url === url && 'focus' in client) return client.focus();
			}
			if (clients.openWindow) return clients.openWindow(url);
		})
	);
});
```

- [ ] **Step 2: Whitelist the source PNGs in `.gitignore`**

The global `*.png` ignore would skip the new source icons. Append to `.gitignore`:
```
!frontend/static/icon-192.png
!frontend/static/icon-512.png
!frontend/static/icon-maskable-192.png
!frontend/static/icon-maskable-512.png
!frontend/static/apple-touch-icon.png
```

- [ ] **Step 3: Register the service worker on mount**

In `frontend/src/routes/+layout.svelte`, inside the `onMount` callback (browser-only), add:
```javascript
if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
```

- [ ] **Step 4: Build and verify assets land in the build root**

Run: `cd frontend && npm run build && ls ../web/build`
Expected: `sw.js`, `manifest.json`, `logo.svg`, and all icon PNGs are present at `web/build/` root.

- [ ] **Step 5: Commit**

```bash
cd /home/zeno/dev/go-stop && git add frontend .gitignore && git commit -m "feat(frontend): ship PWA assets + register service worker"
```

---

### Task 30: Migrate the Playwright e2e suite

**Files:**
- Modify: `playwright.config.js`
- Modify: `e2e/gostop.spec.js`

> Goal: all 32 tests pass against the Go server serving the SvelteKit build on `:8080`. The DOM hooks were preserved per Appendix C; the changes here are (a) replacing the 7 `window.renderX()` calls with real navigation (D3), (b) the localStorage encoding shift for the profile keys (D2), and (c) any selector that genuinely changed.

- [ ] **Step 1: webServer that builds the frontend and serves it via Go**

`playwright.config.js`:
```javascript
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
	testDir: './e2e',
	timeout: 30000,
	use: {
		baseURL: 'http://localhost:8080',
		headless: true,
		timezoneId: 'Europe/Paris'
	},
	reporter: [['list'], ['html', { open: 'never' }]],
	webServer: {
		// Builds the SvelteKit app into web/build, then runs the Go server that serves it.
		// Requires Postgres + VAPID env (e.g. `docker-compose up -d db migrations` first,
		// or an already-running stack — reuseExistingServer picks that up).
		command: 'npm run build && go run .',
		url: 'http://localhost:8080',
		reuseExistingServer: true,
		timeout: 180000
	}
});
```
> If your dev stack already runs on `:8080` (via `docker-compose up` or `make dev`), `reuseExistingServer: true` uses it and skips the command. Otherwise ensure `DATABASE_URL`, `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL` are exported so `go run .` can boot.

- [ ] **Step 2: Fix the profile-seeding helper for the persisted-store encoding (D2)**

In `e2e/gostop.spec.js`, update `setProfile` so `user_name`/`user_phone` are JSON-encoded (matching `svelte-persisted-store`), while `lang` stays a raw string (the Paraglide `lang` strategy reads it raw):
```javascript
async function setProfile(page, user) {
	await page.addInitScript((u) => {
		localStorage.setItem('user_name', JSON.stringify(u.name));
		localStorage.setItem('user_phone', JSON.stringify(u.phone));
		localStorage.setItem('lang', 'fr');
	}, user);
}
```
(If the suite uses `setFr`/`gotoFr` that set `localStorage.lang`, keep them setting the raw string `'fr'`.)

- [ ] **Step 3: Update test 30's assertion to read through the JSON encoding**

In test 30 (`Me page saves name and phone to localStorage`), change the localStorage assertions to:
```javascript
expect(JSON.parse(await page.evaluate(() => localStorage.getItem('user_name')))).toBe('Marie');
expect(JSON.parse(await page.evaluate(() => localStorage.getItem('user_phone')))).toBe('0644000001');
```

- [ ] **Step 4: Replace the 7 `window.renderX()` calls with real navigation (D3)**

Apply these replacements in `e2e/gostop.spec.js` (each was previously `await page.evaluate(() => renderX(...))`):

| Test | Old call | New navigation |
|---|---|---|
| 7 | `renderMyRides()` | `await page.goto('/my-rides'); await page.waitForSelector('.card');` |
| 8 | `renderNotifyRoute('Saillans','Crest')` | `await page.goto('/post-request?origin=Saillans&destination=Crest'); await page.waitForSelector('#notify-form');` |
| 9 | `renderMySearches()` | `await page.goto('/my-searches'); await page.waitForSelector('#my-searches-form');` |
| 10 | `renderPostRide()` | `await page.goto('/post-ride'); await page.waitForSelector('input[name=departure_at]');` |
| 29 | `renderMyRides()` | `await page.goto('/my-rides');` then the existing `#my-rides-form` submit + `.btn-ping-searcher` wait |
| 31 | `renderMe()` | `await page.goto('/me'); await page.waitForSelector('#me-form');` |
| 32 | `renderMe()` | `await page.goto('/me'); await page.waitForSelector('#me-form');` |

For tests 7, 9, 29 the page auto-submits the gate form when a profile phone is present (set via `setProfile`); if a test does not set a profile first, keep its explicit form-fill + submit. For test 10, after navigation set `input[name=departure_at]` then click `#btn-return` and assert `input[name=return_departure_at]` as before.

- [ ] **Step 5: Run the full suite; iteratively fix any selector that genuinely changed**

Run: `cd /home/zeno/dev/go-stop && npm run test:e2e`
Expected: 32 passed. If a test fails on a selector, reconcile against **Appendix C** — the hook should exist; if a component is missing a required id/class/`name`, add it to that component (it is a contract violation, not a test bug). Only change the test when the legacy selector has no Appendix-C entry (e.g. a structural `nth` that the new layout legitimately reorders). Re-run until green.

- [ ] **Step 6: Commit**

```bash
cd /home/zeno/dev/go-stop && git add playwright.config.js e2e/gostop.spec.js && git commit -m "test(e2e): run against SvelteKit build; navigate routes instead of render globals"
```

---

### Task 31: Delete the legacy frontend + final verification

**Files:**
- Delete: `web/js/app.js`, `web/css/style.css`, `web/index.html`, `web/js/sw.js`, and the now-duplicated `web/manifest.json`, `web/logo.svg`, `web/*.png`, `web/icon-maskable.svg`
- Modify: `.gitignore` (drop the obsolete `!web/icon-*.png` / `!web/apple-touch-icon.png` whitelist lines)

- [ ] **Step 1: Remove the legacy files**

Run:
```bash
cd /home/zeno/dev/go-stop
git rm web/js/app.js web/css/style.css web/index.html web/js/sw.js
git rm web/manifest.json web/logo.svg web/icon-maskable.svg
git rm web/icon-192.png web/icon-512.png web/icon-maskable-192.png web/icon-maskable-512.png web/apple-touch-icon.png
```
(The canonical copies now live in `frontend/static/`. `web/build/` remains, gitignored.)

- [ ] **Step 2: Drop obsolete gitignore whitelist lines**

In `.gitignore`, remove the now-dangling `!web/icon-192.png`, `!web/icon-512.png`, `!web/icon-maskable-192.png`, `!web/icon-maskable-512.png`, `!web/apple-touch-icon.png` lines (their targets are deleted; the `frontend/static/*` whitelists from Task 29 replace them).

- [ ] **Step 3: TypeScript strict check (zero `any`, zero errors)**

Run: `cd frontend && npm run check`
Expected: `svelte-check` reports 0 errors, 0 warnings. Fix any type errors before proceeding.

- [ ] **Step 4: Full unit suite + Paraglide key parity**

Run: `cd frontend && npm run paraglide && npm run test:unit -- --run`
Expected: all unit tests pass, including `messages.test.ts` (zero missing translation keys across all 6 locales).

- [ ] **Step 5: Production build from a clean tree**

Run: `cd /home/zeno/dev/go-stop && rm -rf web/build && npm run build && ls web/build/index.html`
Expected: build succeeds; `web/build/index.html` exists.

- [ ] **Step 6: Go build + Go tests**

Run: `cd /home/zeno/dev/go-stop && go build ./... && go test ./... -count=1`
Expected: compiles; tests pass (includes `TestSPAFallbackServesIndex`).

- [ ] **Step 7: Full e2e against the build**

Ensure the DB stack is available, then run: `cd /home/zeno/dev/go-stop && npm run test:e2e`
Expected: **32 passed**.

- [ ] **Step 8: Manual PWA smoke check**

With the server running (`make dev` or compose), load `http://localhost:8080`, confirm: home renders with feed/stats; language switch persists across reload; post a ride → lands on My Rides; service worker registers (DevTools → Application → Service Workers shows `/sw.js`); `manifest.json` loads. Stop the server.

- [ ] **Step 9: Final commit**

```bash
cd /home/zeno/dev/go-stop
git add -A
git commit -m "chore: remove legacy vanilla-JS frontend; SvelteKit refactor complete"
```

---

## Acceptance criteria (from the spec) — final checklist

Run these to confirm every spec acceptance item:

- [ ] `make dev` starts Go (8080) + Vite (5173); app works at `http://localhost:5173` (proxying `/api`).
- [ ] `npm run build` (root) produces `web/build/` with no errors.
- [ ] Go binary serves `web/build/` and all `/api` routes work (`TestSPAFallbackServesIndex` + manual).
- [ ] All 32 Playwright e2e tests pass (`npm run test:e2e`).
- [ ] Vitest unit tests pass with ≥1 test per foundation module + key components (`npm run test:unit -- --run`).
- [ ] TypeScript strict: `npm run check` → 0 errors.
- [ ] Paraglide: `messages.test.ts` green → zero missing keys across 6 locales.
- [ ] PWA: push subscribe, A2HS flow, standalone detection, poll toast preserved (manual smoke).
- [ ] Scalingo deploy succeeds with `.buildpacks` (nodejs → go) — verify on next deploy or `scalingo --app … deploy`.

---

## Self-review (performed by the plan author)

**Spec coverage:** Every spec section maps to tasks — Tech stack (Tasks 1,5,6), project structure (Tasks 1–2, file map), routes (Tasks 19–28 cover all 11 + 2 deep links + 2 redirects), component architecture (Tasks 11–18 cover all listed components; `PageBar` folded into the layout header with a conditional `#back`, noted in Task 11), shared state (Task 8 + D2 deviation), API layer (Tasks 9 + Appendix A), PWA/push (Tasks 10,18,29), build pipeline (Tasks 3,4), testing (Tasks 6–28 unit + Task 30 e2e), migration notes (Task 31 deletes `app.js`/`style.css`/`index.html`; `esc()` replaced by Svelte escaping; `formatTime`/`formatDate` in `utils.ts`; i18n fully migrated in Appendix B). Acceptance criteria enumerated above.

**Known deviations** are documented in **Decisions & Deviations** (D1–D8) with rationale; the most consequential are D1 (e2e on :8080), D2 (localStorage keys preserved, not a `profile` object), D3/D4 (global-fn tests → route navigation + `/post-request` route), and D6 (Paraglide `lang` persistence).

**Type/name consistency:** API field casing is centralized in Appendix A and reused everywhere (`ride.DriverName`, `interest.searcher_name`, `stat.Origin`). Message keys are defined once (Appendix B) and referenced as `m.<key>()`; the parity test (Task 6) guards against drift. localStorage keys are centralized in Appendix D and the Test-Hook Contract (Appendix C) is the single source for every e2e selector.

**Residual risks flagged for the executor:**
1. **Paraglide custom-strategy API** (Task 6, Step 7/9) — `defineCustomClientStrategy` vs. the `overwriteGetLocale/overwriteSetLocale` fallback; the persistence test forces the correct one.
2. **shadcn-svelte / Tailwind v4 init** (Task 5) — the CLI is interactive; accept the defaulted aliases so `$lib/components/ui` and `$lib/utils` resolve.
3. **e2e DB dependency** (Task 30) — `webServer` reuses an existing `:8080` stack; ensure Postgres + VAPID env when none is running.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-06-02-frontend-refactor.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration. REQUIRED SUB-SKILL: superpowers:subagent-driven-development.

**2. Inline Execution** — execute tasks in this session using superpowers:executing-plans, batch execution with checkpoints.

**Which approach?**
