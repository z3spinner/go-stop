# Printable Flyer Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a print-optimised awareness flyer at `/flyer` that renders the deployment's site name, a host-derived URL + QR code, and any of the app's 7 languages via an on-page picker.

**Architecture:** A new prerendered SvelteKit route holds a self-contained "travel-poster" component. Host-dependent values (URL, QR) resolve client-side in `onMount`. The language picker renders copy via Paraglide's per-call `{ locale }` option, so it never reloads or changes the app's own locale. The root layout drops its chrome on `/flyer` for a clean canvas, and entry points are added in the footer and the About page.

**Tech Stack:** SvelteKit (Svelte 5 runes), Paraglide (inlang) i18n, `qrcode` npm package, Vitest + @testing-library/svelte.

---

## File structure

**Create**
- `frontend/src/routes/flyer/+page.ts` — prerender flags.
- `frontend/src/routes/flyer/+page.svelte` — the poster, language picker, print button, QR. One responsibility: the flyer.
- `frontend/src/routes/flyer.test.ts` — component behaviour tests.

**Modify**
- `frontend/package.json` — add `qrcode` + `@types/qrcode`.
- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` — flyer message keys.
- `frontend/src/routes/+layout.svelte` — `isFlyer` branch (drop chrome) + footer `/flyer` link.
- `frontend/src/routes/about/+page.svelte` — closing link to `/flyer`.
- `frontend/src/routes/about/about.test.ts` *(or inline in an existing about test if present — otherwise create)* — assert the flyer link renders.

All commands below run from the `frontend/` directory unless stated.

---

## Task 1: Dependencies and message keys

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/src/messages/fr.json`, `en.json`, `es.json`, `it.json`, `de.json`, `nl.json`, `el.json`

- [ ] **Step 1: Install the QR dependency**

Run:
```bash
cd frontend
npm install qrcode
npm install -D @types/qrcode
```
Expected: `qrcode` appears under `dependencies` and `@types/qrcode` under `devDependencies` in `package.json`.

- [ ] **Step 2: Add the flyer keys to every locale file**

Each file in `src/messages/` is a flat JSON object beginning with `"$schema"`. Add the following key/value pairs to each file (insert before the closing `}`, keeping valid JSON — add a comma after the previous last entry). `{siteName}` is an inlang placeholder; keep it literally.

**`fr.json`** (base locale):
```json
"flyerMetaTitle": "Affiche",
"flyerMetaDescription": "Affiche imprimable pour faire connaître {siteName} autour de vous.",
"flyerHeadlineLine1": "Besoin d'un trajet ?",
"flyerHeadlineLine2": "Offrez une place.",
"flyerSubtitle": "Du covoiturage entre voisins, près de chez nous. Partagez la route, tout simplement. Sans appli, sans compte.",
"flyerStep1Title": "Publiez ou cherchez",
"flyerStep1Desc": "un trajet ponctuel",
"flyerStep2Title": "Soyez prévenu",
"flyerStep2Desc": "dès qu'il y a une correspondance",
"flyerStep3Title": "Appelez & partez",
"flyerStep3Desc": "contact direct",
"flyerCtaLabel": "Rejoignez-nous sur",
"flyerScan": "scanner",
"flyerBadgeLocal": "Local",
"flyerBadgeNoApp": "Sans appli",
"flyerBadgeNoAccount": "Sans compte",
"flyerBadgeFree": "Gratuit",
"flyerPinMe": "à afficher",
"flyerPrint": "Imprimer",
"flyerAboutLink": "Imprimez une affiche à partager près de chez vous"
```

**`en.json`:**
```json
"flyerMetaTitle": "Flyer",
"flyerMetaDescription": "Printable flyer to help spread the word about {siteName}.",
"flyerHeadlineLine1": "Need a lift?",
"flyerHeadlineLine2": "Offer a seat.",
"flyerSubtitle": "Neighbourly ride-sharing for our area. Share the drive, just go together. No app, no account.",
"flyerStep1Title": "Post or search",
"flyerStep1Desc": "a one-off trip",
"flyerStep2Title": "Get notified",
"flyerStep2Desc": "the moment it matches",
"flyerStep3Title": "Call & go",
"flyerStep3Desc": "contact direct",
"flyerCtaLabel": "Hop on at",
"flyerScan": "scan",
"flyerBadgeLocal": "Local",
"flyerBadgeNoApp": "No app",
"flyerBadgeNoAccount": "No account",
"flyerBadgeFree": "Free",
"flyerPinMe": "pin me up",
"flyerPrint": "Print",
"flyerAboutLink": "Print a flyer to share locally"
```

**`es.json`:**
```json
"flyerMetaTitle": "Cartel",
"flyerMetaDescription": "Cartel imprimible para dar a conocer {siteName} en tu zona.",
"flyerHeadlineLine1": "¿Necesitas que te lleven?",
"flyerHeadlineLine2": "Ofrece un asiento.",
"flyerSubtitle": "Coche compartido entre vecinos, aquí al lado. Comparte el viaje, sin más. Sin app, sin cuenta.",
"flyerStep1Title": "Publica o busca",
"flyerStep1Desc": "un viaje puntual",
"flyerStep2Title": "Recibe un aviso",
"flyerStep2Desc": "en cuanto haya coincidencia",
"flyerStep3Title": "Llama y en marcha",
"flyerStep3Desc": "contacto directo",
"flyerCtaLabel": "Únete en",
"flyerScan": "escanear",
"flyerBadgeLocal": "Local",
"flyerBadgeNoApp": "Sin app",
"flyerBadgeNoAccount": "Sin cuenta",
"flyerBadgeFree": "Gratis",
"flyerPinMe": "para colgar",
"flyerPrint": "Imprimir",
"flyerAboutLink": "Imprime un cartel para compartir en tu zona"
```

**`it.json`:**
```json
"flyerMetaTitle": "Volantino",
"flyerMetaDescription": "Volantino stampabile per far conoscere {siteName} nella tua zona.",
"flyerHeadlineLine1": "Ti serve un passaggio?",
"flyerHeadlineLine2": "Offri un posto.",
"flyerSubtitle": "Condivisione di viaggi tra vicini, qui vicino. Condividi la strada, semplicemente. Senza app, senza account.",
"flyerStep1Title": "Pubblica o cerca",
"flyerStep1Desc": "un viaggio occasionale",
"flyerStep2Title": "Ricevi un avviso",
"flyerStep2Desc": "appena c'è una corrispondenza",
"flyerStep3Title": "Chiama e parti",
"flyerStep3Desc": "contatto diretto",
"flyerCtaLabel": "Unisciti su",
"flyerScan": "scansiona",
"flyerBadgeLocal": "Locale",
"flyerBadgeNoApp": "Senza app",
"flyerBadgeNoAccount": "Senza account",
"flyerBadgeFree": "Gratis",
"flyerPinMe": "da appendere",
"flyerPrint": "Stampa",
"flyerAboutLink": "Stampa un volantino da condividere nella tua zona"
```

**`de.json`:**
```json
"flyerMetaTitle": "Aushang",
"flyerMetaDescription": "Druckbarer Aushang, um {siteName} in deiner Umgebung bekannt zu machen.",
"flyerHeadlineLine1": "Brauchst du eine Mitfahrt?",
"flyerHeadlineLine2": "Biete einen Platz an.",
"flyerSubtitle": "Nachbarschaftliche Mitfahrten bei uns vor Ort. Teilt einfach die Fahrt. Keine App, kein Konto.",
"flyerStep1Title": "Einstellen oder suchen",
"flyerStep1Desc": "eine einzelne Fahrt",
"flyerStep2Title": "Benachrichtigt werden",
"flyerStep2Desc": "sobald es passt",
"flyerStep3Title": "Anrufen & los",
"flyerStep3Desc": "direkter Kontakt",
"flyerCtaLabel": "Mitmachen unter",
"flyerScan": "scannen",
"flyerBadgeLocal": "Lokal",
"flyerBadgeNoApp": "Keine App",
"flyerBadgeNoAccount": "Kein Konto",
"flyerBadgeFree": "Kostenlos",
"flyerPinMe": "zum Aushängen",
"flyerPrint": "Drucken",
"flyerAboutLink": "Drucke einen Aushang für deine Umgebung"
```

**`nl.json`:**
```json
"flyerMetaTitle": "Flyer",
"flyerMetaDescription": "Printbare flyer om {siteName} in de buurt bekend te maken.",
"flyerHeadlineLine1": "Zoek je een lift?",
"flyerHeadlineLine2": "Bied een plek aan.",
"flyerSubtitle": "Samen rijden met buren, hier dichtbij. Deel gewoon de rit. Geen app, geen account.",
"flyerStep1Title": "Plaats of zoek",
"flyerStep1Desc": "een eenmalige rit",
"flyerStep2Title": "Krijg een melding",
"flyerStep2Desc": "zodra er een match is",
"flyerStep3Title": "Bel & ga",
"flyerStep3Desc": "direct contact",
"flyerCtaLabel": "Doe mee op",
"flyerScan": "scannen",
"flyerBadgeLocal": "Lokaal",
"flyerBadgeNoApp": "Geen app",
"flyerBadgeNoAccount": "Geen account",
"flyerBadgeFree": "Gratis",
"flyerPinMe": "om op te hangen",
"flyerPrint": "Afdrukken",
"flyerAboutLink": "Druk een flyer af om in de buurt te delen"
```

**`el.json`:**
```json
"flyerMetaTitle": "Αφίσα",
"flyerMetaDescription": "Εκτυπώσιμη αφίσα για να γνωρίσουν το {siteName} στην περιοχή σου.",
"flyerHeadlineLine1": "Χρειάζεσαι μεταφορά;",
"flyerHeadlineLine2": "Πρόσφερε μια θέση.",
"flyerSubtitle": "Κοινή μετακίνηση μεταξύ γειτόνων, εδώ κοντά. Μοιραστείτε τη διαδρομή, απλά. Χωρίς εφαρμογή, χωρίς λογαριασμό.",
"flyerStep1Title": "Δημοσίευσε ή ψάξε",
"flyerStep1Desc": "μια μεμονωμένη διαδρομή",
"flyerStep2Title": "Ειδοποιήσου",
"flyerStep2Desc": "μόλις υπάρξει αντιστοίχιση",
"flyerStep3Title": "Κάλεσε & ξεκίνα",
"flyerStep3Desc": "άμεση επικοινωνία",
"flyerCtaLabel": "Έλα στο",
"flyerScan": "σάρωσε",
"flyerBadgeLocal": "Τοπικά",
"flyerBadgeNoApp": "Χωρίς εφαρμογή",
"flyerBadgeNoAccount": "Χωρίς λογαριασμό",
"flyerBadgeFree": "Δωρεάν",
"flyerPinMe": "για ανάρτηση",
"flyerPrint": "Εκτύπωση",
"flyerAboutLink": "Τύπωσε μια αφίσα για να τη μοιραστείς στην περιοχή σου"
```

- [ ] **Step 3: Recompile messages and verify JSON is valid**

Run:
```bash
npm run paraglide
```
Expected: compile succeeds with no errors and regenerates `src/lib/paraglide/messages/`. If any locale file has a JSON syntax error (trailing/missing comma), the compile fails — fix and re-run.

- [ ] **Step 4: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/messages frontend/src/lib/paraglide
git commit -m "feat(flyer): add qrcode dep and flyer message keys (7 locales)"
```

---

## Task 2: The flyer route component

**Files:**
- Create: `frontend/src/routes/flyer/+page.ts`
- Create: `frontend/src/routes/flyer/+page.svelte`
- Test: `frontend/src/routes/flyer.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/routes/flyer.test.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import { config } from '$lib/config';

// qrcode is async and DOM-free; stub it so the component renders deterministically.
vi.mock('qrcode', () => ({
	default: { toString: vi.fn().mockResolvedValue('<svg data-testid="qr-svg"></svg>') }
}));

import Flyer from './flyer/+page.svelte';

beforeEach(() => {
	config.set({ siteName: 'Go-Stop Saillans', returnDelayHours: 2 });
});

describe('flyer', () => {
	it('shows the configured site name', () => {
		const { getByText } = render(Flyer);
		expect(getByText('Go-Stop Saillans')).toBeTruthy();
	});

	it('shows the host derived from the page origin', async () => {
		const { getByText } = render(Flyer);
		await waitFor(() =>
			expect(getByText((t) => t.includes(window.location.host))).toBeTruthy()
		);
	});

	it('switches flyer language on demand', async () => {
		const { getByText } = render(Flyer);
		await fireEvent.click(getByText('EN'));
		expect(getByText('Need a lift?')).toBeTruthy();
		await fireEvent.click(getByText('FR'));
		expect(getByText((t) => t.includes("Besoin d'un trajet"))).toBeTruthy();
	});

	it('print button calls window.print', async () => {
		const printSpy = vi.fn();
		window.print = printSpy;
		const { getByText } = render(Flyer);
		await fireEvent.click(getByText('FR')); // make the print label deterministic ("Imprimer")
		await fireEvent.click(getByText('Imprimer'));
		expect(printSpy).toHaveBeenCalled();
	});
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
npx vitest run src/routes/flyer.test.ts
```
Expected: FAIL — cannot resolve `./flyer/+page.svelte` (file does not exist yet).

- [ ] **Step 3: Create the prerender flags**

Create `frontend/src/routes/flyer/+page.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// Like /about, this page is prerendered to static HTML (SSR on so the
// prerenderer can render it at build time). Host-dependent values (URL, QR)
// are filled in client-side on mount.
export const prerender = true;
export const ssr = true;
```

- [ ] **Step 4: Create the component**

Create `frontend/src/routes/flyer/+page.svelte`:
```svelte
<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
	import { getLocale, locales, baseLocale, type Locale } from '$lib/locale';

	// `selected` drives only the flyer's rendered copy (via Paraglide's per-call
	// { locale } option) — it never touches the app's global locale or reloads.
	let selected = $state<Locale>(baseLocale);
	let origin = $state('');
	let qrSvg = $state('');

	let host = $derived(origin.replace(/^https?:\/\//, ''));

	const localeNames: Record<Locale, string> = {
		fr: 'FR', en: 'EN', es: 'ES', it: 'IT', de: 'DE', nl: 'NL', el: 'EL'
	};

	onMount(async () => {
		selected = getLocale();
		origin = window.location.origin;
		try {
			qrSvg = await QRCode.toString(origin, {
				type: 'svg',
				margin: 0,
				errorCorrectionLevel: 'M'
			});
		} catch {
			/* leave QR empty if generation fails */
		}
	});
</script>

<svelte:head>
	<title>{m.flyerMetaTitle({}, { locale: selected })} · {$config.siteName}</title>
	<meta
		name="description"
		content={m.flyerMetaDescription({ siteName: $config.siteName }, { locale: selected })}
	/>
	<link rel="canonical" href="/flyer" />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
	<link
		href="https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,600;0,9..144,900;1,9..144,600;1,9..144,800&family=Bricolage+Grotesque:wght@500;600;700;800&family=Space+Mono:wght@700&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="flyer-controls no-print">
	<div class="lang-row">
		{#each locales as l}
			<button
				type="button"
				class="lang-btn"
				class:active={l === selected}
				aria-pressed={l === selected}
				onclick={() => (selected = l)}>{localeNames[l]}</button
			>
		{/each}
	</div>
	<button type="button" class="print-btn" onclick={() => window.print()}>
		{m.flyerPrint({}, { locale: selected })}
	</button>
</div>

<div class="flyer-stage">
	<div class="poster">
		<span class="pin l"></span><span class="pin r"></span>

		<div class="brand">
			<svg class="mark" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
				<defs>
					<clipPath id="flyerClip"><rect width="512" height="512" rx="115" /></clipPath>
					<linearGradient id="flyerGl" x1="0" y1="0" x2="0.8" y2="1">
						<stop offset="0" stop-color="#4CD964" /><stop offset="1" stop-color="#28A836" />
					</linearGradient>
					<linearGradient id="flyerGr" x1="0.2" y1="0" x2="1" y2="1">
						<stop offset="0" stop-color="#FF4538" /><stop offset="1" stop-color="#C81E1E" />
					</linearGradient>
				</defs>
				<g clip-path="url(#flyerClip)">
					<rect width="256" height="512" fill="url(#flyerGl)" />
					<rect x="256" width="256" height="512" fill="url(#flyerGr)" />
				</g>
				<g fill="none" stroke="#fff" stroke-width="22" stroke-linecap="round" stroke-linejoin="round">
					<path d="M 196,175 A 106,106 0 1 0 234,256 L 278,256" /><circle cx="384" cy="256" r="106" />
				</g>
				<polygon points="90,196 90,316 198,256" fill="#fff" />
				<rect x="330" y="202" width="108" height="108" rx="20" fill="#fff" />
			</svg>
			<span class="wordmark">{$config.siteName}</span>
		</div>

		<h1 class="hero">
			{m.flyerHeadlineLine1({}, { locale: selected })}<em>{m.flyerHeadlineLine2({}, { locale: selected })}</em>
		</h1>
		<p class="sub">{m.flyerSubtitle({}, { locale: selected })}</p>

		<div class="route">
			<div class="stop">
				<div class="num">1</div>
				<b>{m.flyerStep1Title({}, { locale: selected })}</b>
				<span>{m.flyerStep1Desc({}, { locale: selected })}</span>
			</div>
			<div class="stop">
				<div class="num">2</div>
				<b>{m.flyerStep2Title({}, { locale: selected })}</b>
				<span>{m.flyerStep2Desc({}, { locale: selected })}</span>
			</div>
			<div class="stop">
				<div class="num">3</div>
				<b>{m.flyerStep3Title({}, { locale: selected })}</b>
				<span>{m.flyerStep3Desc({}, { locale: selected })}</span>
			</div>
		</div>

		<div class="dest">
			<div class="main">
				<span class="label">{m.flyerCtaLabel({}, { locale: selected })}</span>
				<span class="url">{host}</span>
			</div>
			<div class="qr">
				<div class="qrimg">{@html qrSvg}</div>
				<span>{m.flyerScan({}, { locale: selected })}</span>
			</div>
		</div>

		<div class="foot">
			<div class="pillrow">
				<span class="pill">{m.flyerBadgeLocal({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeNoApp({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeNoAccount({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeFree({}, { locale: selected })}</span>
			</div>
			<span class="tear">♺ {m.flyerPinMe({}, { locale: selected })}</span>
		</div>
	</div>
</div>

<style>
	/* ---- screen controls (never printed) ---- */
	.flyer-controls {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 14px;
		flex-wrap: wrap;
		padding: 16px;
	}
	.lang-row { display: flex; gap: 6px; flex-wrap: wrap; }
	.lang-btn {
		font: 700 12px/1 system-ui, sans-serif;
		padding: 6px 10px;
		border: 1px solid #cfd8cf;
		border-radius: 8px;
		background: #fff;
		color: #2b332b;
		cursor: pointer;
	}
	.lang-btn.active { background: #1f9d3a; border-color: #1f9d3a; color: #fff; }
	.print-btn {
		font: 700 13px/1 system-ui, sans-serif;
		padding: 9px 18px;
		border: none;
		border-radius: 999px;
		background: #1f9d3a;
		color: #fff;
		cursor: pointer;
	}

	.flyer-stage { display: flex; justify-content: center; padding: 0 16px 32px; }

	/* ---- the poster (A4 portrait, ~720x1018 px ≈ 190x269 mm) ---- */
	.poster {
		--green: #1f9d3a; --green-deep: #15772b; --ink: #1f201a; --paper: #fbf6ea;
		position: relative;
		width: 720px;
		height: 1018px;
		background: var(--paper);
		color: var(--ink);
		border-radius: 6px;
		overflow: hidden;
		padding: 60px 64px 46px;
		box-shadow: 0 18px 50px rgba(40, 30, 10, 0.28);
		display: flex;
		flex-direction: column;
		font-family: 'Bricolage Grotesque', sans-serif;
		print-color-adjust: exact;
		-webkit-print-color-adjust: exact;
	}
	.poster::before {
		content: '';
		position: absolute;
		inset: 0;
		pointer-events: none;
		background:
			radial-gradient(120% 80% at 50% -10%, #ffffff 0%, transparent 55%),
			radial-gradient(140% 100% at 50% 120%, rgba(180, 150, 90, 0.18) 0%, transparent 60%);
	}
	.poster::after {
		content: '';
		position: absolute;
		inset: 0;
		pointer-events: none;
		opacity: 0.45;
		mix-blend-mode: multiply;
		background-image: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='120' height='120'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/><feColorMatrix type='saturate' values='0'/></filter><rect width='100%25' height='100%25' filter='url(%23n)' opacity='0.5'/></svg>");
	}
	.poster > * { position: relative; z-index: 1; }

	.pin {
		position: absolute;
		width: 19px;
		height: 19px;
		border-radius: 50%;
		background: radial-gradient(circle at 35% 30%, #fff 0 18%, #cf3b2e 22% 60%, #8e231a 100%);
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.35);
		z-index: 2;
	}
	.pin.l { top: 22px; left: 22px; }
	.pin.r { top: 22px; right: 22px; }

	.brand { display: flex; align-items: center; gap: 18px; }
	.mark { width: 64px; height: 64px; border-radius: 16px; box-shadow: 0 3px 10px rgba(0, 0, 0, 0.18); flex: none; }
	.wordmark { font-weight: 800; font-size: 36px; letter-spacing: -0.02em; }

	.hero {
		font-family: 'Fraunces', serif;
		font-weight: 900;
		font-size: 92px;
		line-height: 0.92;
		letter-spacing: -0.02em;
		margin: 42px 0 0;
	}
	.hero em { font-style: italic; font-weight: 800; color: var(--green); display: block; }

	.sub {
		font-family: 'Fraunces', serif;
		font-style: italic;
		font-weight: 600;
		font-size: 26px;
		line-height: 1.35;
		color: #5b5a4e;
		margin-top: 24px;
		max-width: 540px;
	}

	.route { display: flex; justify-content: space-between; position: relative; margin: 44px 6px 0; }
	.route::before {
		content: '';
		position: absolute;
		top: 27px;
		left: 8%;
		right: 8%;
		border-top: 4px dashed #c9bfa3;
		z-index: 0;
	}
	.stop {
		position: relative;
		z-index: 1;
		width: 31%;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
	}
	.num {
		width: 55px;
		height: 55px;
		border-radius: 50%;
		background: var(--green);
		color: #fff;
		font-weight: 800;
		font-size: 27px;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 5px 14px rgba(31, 157, 58, 0.4);
		border: 5px solid var(--paper);
	}
	.stop b { font-size: 21px; font-weight: 700; line-height: 1.15; }
	.stop span { font-size: 18px; color: #6c6a5d; line-height: 1.2; }

	.dest {
		margin: 46px 0 0;
		display: flex;
		align-items: stretch;
		border-radius: 28px;
		overflow: hidden;
		box-shadow: 0 12px 28px rgba(21, 119, 43, 0.34);
	}
	.dest .main {
		flex: 1;
		min-width: 0;
		background: linear-gradient(155deg, var(--green) 0%, var(--green-deep) 100%);
		color: #fff;
		padding: 32px 36px;
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 8px;
	}
	.dest .label {
		font-size: 16px;
		font-weight: 700;
		letter-spacing: 0.24em;
		text-transform: uppercase;
		opacity: 0.85;
	}
	.dest .url {
		font-family: 'Space Mono', monospace;
		font-weight: 700;
		font-size: 40px;
		letter-spacing: -0.03em;
		line-height: 1.05;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.dest .qr {
		background: #fff;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 25px;
		flex: none;
	}
	.dest .qrimg { width: 116px; height: 116px; }
	.dest .qrimg :global(svg) { width: 100%; height: 100%; display: block; }
	.dest .qr span {
		font-size: 14px;
		font-weight: 700;
		letter-spacing: 0.16em;
		color: #6c6a5d;
		text-transform: uppercase;
	}

	/* margin-top:auto on the footer pools any slack BELOW the badges, keeping the
	   URL/destination box up as the focal point (the flagged polish item). */
	.foot { display: flex; align-items: center; gap: 12px; margin-top: auto; padding-top: 28px; }
	.pillrow { display: flex; gap: 10px; flex-wrap: wrap; }
	.pill {
		font-size: 17px;
		font-weight: 700;
		color: var(--green-deep);
		background: #e9f6ea;
		border-radius: 999px;
		padding: 7px 19px;
	}
	.tear { margin-left: auto; font-size: 17px; font-style: italic; font-family: 'Fraunces', serif; color: #9a9685; }

	/* ---- print ---- */
	@media print {
		:global(.no-print) { display: none !important; }
		.flyer-stage { padding: 0; }
		.poster { box-shadow: none; border-radius: 0; }
	}
</style>
```

Then add the A4 page rule. Append to `frontend/src/app.css` (global; `@page` cannot be component-scoped):
```css
@page {
	size: A4 portrait;
	margin: 12mm;
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run:
```bash
npx vitest run src/routes/flyer.test.ts
```
Expected: PASS (4 tests). If "qrcode" fails to resolve, confirm Task 1 installed it.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/routes/flyer frontend/src/routes/flyer.test.ts frontend/src/app.css
git commit -m "feat(flyer): add printable /flyer poster with language picker and QR"
```

---

## Task 3: Strip app chrome on /flyer and add the footer link

**Files:**
- Modify: `frontend/src/routes/+layout.svelte`

- [ ] **Step 1: Add the `isFlyer` derived flag**

In `frontend/src/routes/+layout.svelte`, find:
```svelte
	let isHome = $derived(page.url.pathname === '/');
```
Add directly below it:
```svelte
	let isFlyer = $derived(page.url.pathname === '/flyer');
```

- [ ] **Step 2: Branch the layout so /flyer renders chrome-free**

Replace the existing markup block:
```svelte
	<header class="top-bar mx-auto flex max-w-xl items-center gap-2 p-3" class:page-bar={!isHome}>
		{#if !isHome}
			<button id="back" type="button" class="btn-back" onclick={back}>{m.btnBack()}</button>
		{/if}
		<TopBar onprivacy={() => (showPrivacy = true)} />
	</header>

	<div id="app" class="mx-auto max-w-xl p-3">
		{#key refreshNonce}
			{@render children()}
		{/key}
	</div>

	<footer id="app-footer" class="mx-auto max-w-xl p-3 text-center text-sm text-gray-500">
		<button type="button" class="btn-footer-privacy underline" onclick={() => (showPrivacy = true)}>{m.footerPrivacy()}</button>
		<span> · </span>
		<a class="btn-footer-about underline" href="/about">{m.aboutTitle()}</a>
		<span> · </span>
		<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
	</footer>
```
with:
```svelte
	{#if isFlyer}
		{#key refreshNonce}
			{@render children()}
		{/key}
	{:else}
		<header class="top-bar mx-auto flex max-w-xl items-center gap-2 p-3" class:page-bar={!isHome}>
			{#if !isHome}
				<button id="back" type="button" class="btn-back" onclick={back}>{m.btnBack()}</button>
			{/if}
			<TopBar onprivacy={() => (showPrivacy = true)} />
		</header>

		<div id="app" class="mx-auto max-w-xl p-3">
			{#key refreshNonce}
				{@render children()}
			{/key}
		</div>

		<footer id="app-footer" class="mx-auto max-w-xl p-3 text-center text-sm text-gray-500">
			<button type="button" class="btn-footer-privacy underline" onclick={() => (showPrivacy = true)}>{m.footerPrivacy()}</button>
			<span> · </span>
			<a class="btn-footer-about underline" href="/about">{m.aboutTitle()}</a>
			<span> · </span>
			<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
			<span> · </span>
			<a class="btn-footer-flyer underline" href="/flyer">{m.flyerMetaTitle()}</a>
		</footer>
	{/if}
```

- [ ] **Step 3: Verify the existing layout/serve tests still pass**

Run:
```bash
npx vitest run
```
Expected: PASS — existing suites (`home.test.ts`, `me.test.ts`, `search.test.ts`, `edit-ride.test.ts`, `flyer.test.ts`) all green. Chrome-stripping is verified manually in Task 5 (the layout depends on `$app/state`, the service worker, and stores that aren't unit-mounted here).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/routes/+layout.svelte
git commit -m "feat(flyer): render /flyer chrome-free and link it from the footer"
```

---

## Task 4: Link the flyer from the About page

**Files:**
- Modify: `frontend/src/routes/about/+page.svelte`
- Test: `frontend/src/routes/about.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/routes/about.test.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import About from './about/+page.svelte';

describe('about', () => {
	it('links to the printable flyer', () => {
		const { container } = render(About);
		const link = container.querySelector('a[href="/flyer"]');
		expect(link).toBeTruthy();
	});
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
npx vitest run src/routes/about.test.ts
```
Expected: FAIL — no `a[href="/flyer"]` in the About page.

- [ ] **Step 3: Add the link**

In `frontend/src/routes/about/+page.svelte`, after the existing closing `</div>` (the `modal-body`), add:
```svelte
<p class="mt-4">
	<a class="underline" href="/flyer">{m.flyerAboutLink()}</a>
</p>
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
npx vitest run src/routes/about.test.ts
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/about/+page.svelte frontend/src/routes/about.test.ts
git commit -m "feat(flyer): link the flyer from the About page"
```

---

## Task 5: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Run the whole frontend unit suite**

Run:
```bash
cd frontend
npx vitest run
```
Expected: all suites PASS.

- [ ] **Step 2: Lint and type-check**

Run:
```bash
npm run lint
npm run check
```
Expected: no errors. (If the repo uses different script names, check `package.json` `scripts` and run the equivalent lint/`svelte-check`.)

- [ ] **Step 3: Production build**

Run:
```bash
npm run build
```
Expected: build succeeds and `/flyer` prerenders without error.

- [ ] **Step 4: Manual print-preview check**

Start the dev stack (`docker compose up` from the repo root, or `npm run dev` in `frontend/`), open `http://localhost:5173/flyer`, and confirm:
- Site name, headline, the three route steps, the URL (real host) and a scannable QR all render.
- The language buttons switch the poster copy **without** reloading or changing the app's own language.
- Browser Print preview (Cmd/Ctrl-P) shows a clean A4 page: cream + green print correctly (enable "Background graphics" if your browser needs it), fonts are loaded, controls are hidden, and **the URL/destination box sits as a clear focal point — not jammed at the bottom edge** (the flagged polish item). Adjust the `.foot`/`.dest` spacing if needed and re-check.

- [ ] **Step 5: Final commit (if Step 4 required tweaks)**

```bash
git add -A
git commit -m "fix(flyer): tune print layout spacing"
```

---

## Self-review notes

- **Spec coverage:** route + prerender (T2), client-resolved URL/QR (T2), travel-poster design (T2), no-reload per-call language picker across 7 locales (T1 keys + T2), `qrcode` SVG (T1+T2), A4 print CSS + `print-color-adjust` (T2), chrome strip via `isFlyer` (T3), footer link (T3), About link (T4), message-key table (T1), flagged URL-box rhythm (T2 `.foot { margin-top:auto }` + T5 manual check). All covered.
- **Fonts:** Google Fonts `<link>` scoped to the page head (per approved decision).
- **Type consistency:** `selected: Locale`, `origin`/`host`/`qrSvg` strings, `localeNames: Record<Locale,string>` used consistently; message keys identical between the locale files and the component calls.
