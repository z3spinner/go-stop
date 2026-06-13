# Printable flyer page — design

**Date:** 2026-06-13
**Status:** Approved (design direction); one polish item flagged for implementation.

## Goal

A printable awareness flyer at `/flyer` that any community running Go-Stop can
print, pin to a café/community noticeboard, and use to drive locals to the site.
It is a real in-app page (not a static one-off): it pulls the deployment's own
site name, builds a QR/URL from the host it's served from, and renders in any of
the app's languages via an on-page picker. Print-optimised so the browser's
Print dialog yields a clean A4 page.

## Visual design

Direction approved via the brainstorming visual companion (final mockup
`poster-v6`, saved under `.superpowers/brainstorm/`). Editorial "travel-poster"
aesthetic:

- **Stock:** warm cream paper (`#fbf6ea`) with a subtle SVG grain + warm vignette.
- **Corners:** two red "push-pin" dots (top-left/right) — a noticeboard nod.
- **Header:** the Go-Stop logo (SVG, the real green/red mark) + the **site name**
  as the wordmark (`$config.siteName`, e.g. "Go-Stop Saillans").
- **Headline:** Fraunces serif, heavy — line 1 in ink, line 2 in brand green
  italic (mock copy: *"Need a lift? / Offer a seat."*).
- **Subtitle:** Fraunces italic, muted — one neighbourly sentence.
- **Route motif:** three numbered stops on a dashed line (Post or search → Get
  notified → Call & go), framing "how it works" as a little journey.
- **Destination card:** rounded green gradient panel holding the **URL** in
  Space Mono (the focal CTA), with a **small** QR to its right labelled "scan".
- **Footer:** pill badges (Local · No app · No account · Free) + a handwritten
  "pin me up".
- **Palette:** brand green `#28a836` dominant (deep `#15772b` for depth), the
  logo's red as a spark, warm ink `#1f201a` on cream.
- **Fonts:** Fraunces (display), Bricolage Grotesque (labels/UI), Space Mono
  (URL). Loaded via Google Fonts in the page `<head>`, **scoped to this route**
  (the rest of the app is unaffected). Self-hosting is a possible later
  hardening step; not required for v1.

### Flagged polish item

In the mockup the destination/URL box drifted to the bottom edge (a
`margin-top:auto` artifact). Implementation must give the poster a **deliberate
vertical rhythm** so the URL box reads as a centred focal point, with any slack
pooling below the footer — not above the URL. Verify in an actual print preview,
not just on screen.

## Architecture

### Route & rendering

- New route `src/routes/flyer/+page.svelte` with `src/routes/flyer/+page.ts`
  setting `export const prerender = true; export const ssr = true;` — matching
  the About page pattern (the rest of the app is a client-only SPA).
- Host-dependent values are resolved **client-side** (the page is prerendered
  with no known host, and each deployment has its own domain):
  - `origin = window.location.origin` (in `onMount`).
  - Display URL = host without protocol (e.g. `go-stop.example.org`); QR encodes
    the full `origin`.
  - Site name from the existing `$config` store (`loadConfig()` already runs in
    the root layout's `onMount`).

### Stripping app chrome

The root `+layout.svelte` wraps every page in a header (back button + TopBar),
a `max-w-xl` content column, and a footer, all inside `PullToRefresh`. The flyer
needs a clean full-bleed canvas while keeping the layout's `onMount`
initialisation (service worker, `loadConfig`, locale strategy).

Follow the existing `isHome` pattern: derive `isFlyer = page.url.pathname ===
'/flyer'` and, when true, render `children()` without the header, footer, or the
`max-w-xl p-3` wrapper. Keep the layout's init untouched. App-chrome overlays
(A2HS banner/modal, toasts) are marked `.no-print` so they never appear on
paper even if visible on screen.

### Language picker (no reload, app-locale-independent)

The app's global `setLocale` writes localStorage + reloads — wrong for switching
just the flyer. Instead the flyer keeps local state and uses Paraglide's
**per-call locale option** (confirmed in the compiled output: every message fn
has signature `(inputs?, { locale? })`):

```svelte
let selected = $state(getLocale());           // default to the app's current locale
m.flyerHeadline({}, { locale: selected })     // every flyer string renders in `selected`
```

- Picker = a small row of buttons, one per locale in `locales`
  (fr, en, es, it, de, nl, el), `.no-print` (hidden on paper).
- Switching updates `selected` only — no reload, no change to the app's own
  language, no localStorage write.

### QR generation

- Add the `qrcode` npm package (mature, maintained).
- In `onMount`, generate an **SVG** (crisp for print) encoding `origin`, inject
  into the destination card. SVG over canvas/PNG so it scales cleanly at print
  DPI.
- QR is intentionally small/secondary (per the URL-forward decision); it has an
  accessible label.

### Print CSS

- `@page { size: A4 portrait; margin: 0 }`; the poster sized in **mm** (e.g.
  ~190mm wide) centred on the sheet so screen and print match and Letter paper
  still fits without clipping.
- `print-color-adjust: exact` (+ `-webkit-`) on the poster so the cream/green
  actually print.
- `.no-print` hides the language picker, the Print button, and any app overlays.
- An on-screen **Print** button (`.no-print`) calls `window.print()`.

### Message keys

New keys added to all seven `src/messages/{locale}.json` files (base locale is
**fr**). Implementation supplies translations for every locale (fr/en authored
carefully; es/it/de/nl/el translated to match). Keys:

| key | en | fr (base) |
| --- | --- | --- |
| `flyerMetaTitle` | Flyer | Affiche |
| `flyerMetaDescription` | Printable flyer to share {siteName} locally. | Affiche imprimable pour faire connaître {siteName} autour de vous. |
| `flyerHeadlineLine1` | Need a lift? | Besoin d'un trajet ? |
| `flyerHeadlineLine2` | Offer a seat. | Offrez une place. |
| `flyerSubtitle` | Neighbourly ride-sharing for our area. Share the drive, just go together. No app, no account. | Du covoiturage entre voisins, près de chez nous. Partagez la route, tout simplement. Sans appli, sans compte. |
| `flyerStep1Title` | Post or search | Publiez ou cherchez |
| `flyerStep1Desc` | a one-off trip | un trajet ponctuel |
| `flyerStep2Title` | Get notified | Soyez prévenu |
| `flyerStep2Desc` | the moment it matches | dès qu'il y a une correspondance |
| `flyerStep3Title` | Call & go | Appelez & partez |
| `flyerStep3Desc` | contact direct | contact direct |
| `flyerCtaLabel` | Hop on at | Rejoignez-nous sur |
| `flyerScan` | scan | scanner |
| `flyerBadgeLocal` | Local | Local |
| `flyerBadgeNoApp` | No app | Sans appli |
| `flyerBadgeNoAccount` | No account | Sans compte |
| `flyerBadgeFree` | Free | Gratuit |
| `flyerPinMe` | pin me up | à afficher |
| `flyerPrint` | Print | Imprimer |

(Final wording reviewable on the page; copy is not load-bearing for the
architecture.)

### Discoverability

Recommended: add a small `/flyer` link to the existing footer (alongside
About · Stats). Low-key; the deployer is the primary audience. Easily changed in
review if you'd rather leave it unlinked or surface it elsewhere.

## Testing

- **Component test** (`src/routes/flyer.test.ts`, vitest + Testing Library,
  matching existing route tests): renders the page with a mocked `$config`
  siteName and a stubbed `window.location.origin`; asserts the site name and
  host appear, the language picker switches rendered copy (e.g. fr → en), and
  the Print button calls `window.print`.
- **Print/visual** check is manual: open `/flyer`, Print preview, confirm A4
  fidelity (colours, fonts loaded, URL box centred per the flagged polish item).
- Optional Playwright e2e smoke if it fits the existing `e2e/` suite.

## Out of scope (YAGNI)

- Tear-off tabs (layout B was rejected).
- Multiple flyer templates / theme variants.
- Server-side PDF generation or a download button (browser Print is enough).
- Editable copy / CMS.

## Files

**Create**
- `src/routes/flyer/+page.svelte` — the poster + picker + print button.
- `src/routes/flyer/+page.ts` — `prerender = true; ssr = true`.
- `src/routes/flyer.test.ts` — component test.

**Modify**
- `src/routes/+layout.svelte` — `isFlyer` branch to drop chrome / full-bleed.
- `src/messages/{fr,en,es,it,de,nl,el}.json` — flyer message keys.
- `src/routes/+layout.svelte` footer (or wherever) — optional `/flyer` link.
- `package.json` — add `qrcode` dependency.
