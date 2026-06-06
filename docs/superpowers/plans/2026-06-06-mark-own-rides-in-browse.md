# Mark Own Rides in Browse/Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In the home "Available" feed and `/search` results, clearly mark rides the current user published as their own, replacing the contact button with a "Manage in My rides" link.

**Architecture:** Frontend-only. Identity is the phone in `localStorage`; public rides omit the driver phone, so own rides are matched by **ride ID**. `RideCard` gains an `isOwn` prop (badge + manage link); each list page builds a `Set<string>` of the user's own ride IDs and passes the flag. No backend/DTO/DB changes.

**Tech Stack:** SvelteKit 2 + Svelte 5 runes; Paraglide (inlang) i18n across 7 locales (base `fr`); Vitest + @testing-library/svelte.

---

## File Structure

- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` — two new keys (`yourRide`, `manageInMyRides`).
- `frontend/src/lib/myRides.ts` — **new** helper `loadMyRideIds(phone)` returning a `Set<string>` (mirrors `$lib/contacts.ts`).
- `frontend/src/lib/myRides.test.ts` — **new** unit test for the helper.
- `frontend/src/lib/components/rides/RideCard.svelte` — `isOwn` prop: badge + manage link replacing `ContactOrInterest`.
- `frontend/src/lib/components/rides/RideCard.test.ts` — own-ride rendering tests.
- `frontend/src/routes/+page.svelte` — build `myIds` from the already-fetched own rides, pass `isOwn`.
- `frontend/src/routes/search/+page.svelte` — fetch own ride IDs via the helper, pass `isOwn`.
- `frontend/src/routes/search.test.ts` — own-ride badge in search results.

---

## Task 1: i18n keys (2 keys, 7 locales)

**Files:**
- Modify: `frontend/src/messages/fr.json`, `en.json`, `es.json`, `it.json`, `de.json`, `nl.json`, `el.json`

No test step (data only); verified by Task 3's render test and the paraglide compile.

- [ ] **Step 1: Add the two keys to every locale file**

Add these to each top-level JSON object (place next to other ride keys; keep valid JSON — read each file first to pick a safe anchor and preserve commas).

`fr.json`:
```json
	"yourRide": "Votre trajet",
	"manageInMyRides": "Gérer dans Mes trajets",
```
`en.json`:
```json
	"yourRide": "Your ride",
	"manageInMyRides": "Manage in My rides",
```
`es.json`:
```json
	"yourRide": "Tu viaje",
	"manageInMyRides": "Gestionar en Mis viajes",
```
`it.json`:
```json
	"yourRide": "Il tuo viaggio",
	"manageInMyRides": "Gestisci in I miei viaggi",
```
`de.json`:
```json
	"yourRide": "Deine Fahrt",
	"manageInMyRides": "In Meine Fahrten verwalten",
```
`nl.json`:
```json
	"yourRide": "Jouw rit",
	"manageInMyRides": "Beheren in Mijn ritten",
```
`el.json`:
```json
	"yourRide": "Η διαδρομή σας",
	"manageInMyRides": "Διαχείριση στις διαδρομές μου",
```

- [ ] **Step 2: Validate JSON + compile paraglide**

Run: `cd frontend && for f in src/messages/*.json; do node -e "JSON.parse(require('fs').readFileSync('$f','utf8'))" && echo "$f OK"; done && npm run paraglide`
Expected: every file `OK`; paraglide compiles with no missing-key/parse errors. Confirm both keys in all 7: `grep -l yourRide src/messages/*.json | wc -l` → `7`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/messages
git commit -m "i18n: add yourRide and manageInMyRides keys"
```

---

## Task 2: `loadMyRideIds` helper

**Files:**
- Create: `frontend/src/lib/myRides.ts`
- Test: `frontend/src/lib/myRides.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/lib/myRides.test.ts`:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';

const list = vi.fn();
vi.mock('$lib/api', () => ({ api: { rides: { list } } }));

import { loadMyRideIds } from './myRides';

beforeEach(() => list.mockReset());

describe('loadMyRideIds', () => {
	it('returns an empty set and makes no request when phone is empty', async () => {
		const ids = await loadMyRideIds('');
		expect(ids.size).toBe(0);
		expect(list).not.toHaveBeenCalled();
	});

	it('returns the set of owned ride IDs for a phone', async () => {
		list.mockResolvedValue([{ ID: 'a1' }, { ID: 'b2' }]);
		const ids = await loadMyRideIds('0612345678');
		expect(list).toHaveBeenCalledWith({}, '0612345678');
		expect([...ids].sort()).toEqual(['a1', 'b2']);
	});

	it('returns an empty set on error', async () => {
		list.mockRejectedValue(new Error('network'));
		const ids = await loadMyRideIds('0612345678');
		expect(ids.size).toBe(0);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/myRides.test.ts`
Expected: FAIL — cannot resolve `./myRides`.

- [ ] **Step 3: Create the helper**

Create `frontend/src/lib/myRides.ts`:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '$lib/api';
import type { Ride } from '$lib/types';

/**
 * Returns the set of ride IDs published by this phone. Empty set when no phone is
 * set or the request fails — callers treat "unknown" as "not mine".
 */
export async function loadMyRideIds(phone: string): Promise<Set<string>> {
	if (!phone) return new Set();
	try {
		const mine = (await api.rides.list({}, phone)) as Ride[];
		return new Set(mine.map((r) => r.ID));
	} catch {
		return new Set();
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/lib/myRides.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/myRides.ts frontend/src/lib/myRides.test.ts
git commit -m "feat(rides): add loadMyRideIds helper for own-ride matching"
```

---

## Task 3: `RideCard` own-ride rendering

**Files:**
- Modify: `frontend/src/lib/components/rides/RideCard.svelte`
- Test: `frontend/src/lib/components/rides/RideCard.test.ts`

- [ ] **Step 1: Write the failing tests**

Add these tests inside the `describe('RideCard', ...)` block in `frontend/src/lib/components/rides/RideCard.test.ts`:

```ts
	it('marks an own ride with a badge and a manage link, and hides the contact button', () => {
		const { container } = render(RideCard, { props: { ride, isOwn: true } });
		const badge = container.querySelector('.tag-your-ride') as HTMLElement;
		expect(badge).toBeTruthy();
		expect(badge.textContent).toMatch(/Your ride|Votre trajet/);
		const manage = container.querySelector('a.ride-manage-link') as HTMLAnchorElement;
		expect(manage).toBeTruthy();
		expect(manage.getAttribute('href')).toBe('/my-rides#card-42');
		expect(container.querySelector('.btn-interest')).toBeNull();
	});

	it('renders the contact action and no own-ride badge by default', () => {
		const { container } = render(RideCard, { props: { ride } });
		expect(container.querySelector('.tag-your-ride')).toBeNull();
		expect(container.querySelector('.ride-manage-link')).toBeNull();
		expect(container.querySelector('.btn-interest')).toBeTruthy();
	});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/lib/components/rides/RideCard.test.ts`
Expected: FAIL — no `.tag-your-ride` / `.ride-manage-link`; `isOwn` prop not wired.

- [ ] **Step 3: Add the `isOwn` prop and markup**

Edit `frontend/src/lib/components/rides/RideCard.svelte`. Update the props destructure:

```svelte
	let {
		ride,
		contactPhone,
		showDriver = true,
		isOwn = false
	}: { ride: PublicRide | Ride; contactPhone?: string; showDriver?: boolean; isOwn?: boolean } = $props();
```

In the `.card-meta` row, add the badge as the first child (immediately after the opening `<div class="card-meta ...">`):

```svelte
			{#if isOwn}<span class="tag tag-your-ride rounded bg-green-100 px-1">{m.yourRide()}</span>{/if}
```

Replace the `<ContactOrInterest {ride} {contactPhone} />` line (just before the closing `</div>` of the card) with:

```svelte
	{#if isOwn}
		<a href="/my-rides#card-{ride.ID}" class="ride-manage-link inline-block text-sm underline">{m.manageInMyRides()} →</a>
	{:else}
		<ContactOrInterest {ride} {contactPhone} />
	{/if}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/lib/components/rides/RideCard.test.ts`
Expected: PASS (existing detail-link test + the 2 new ones).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/components/rides/RideCard.svelte frontend/src/lib/components/rides/RideCard.test.ts
git commit -m "feat(rides): mark own rides on RideCard with badge + manage link"
```

---

## Task 4: Home page wiring

**Files:**
- Modify: `frontend/src/routes/+page.svelte`

No new test (covered by RideCard unit test + Task 5 search test + manual smoke); the home page's own-rides fetch already exists.

- [ ] **Step 1: Add `myIds` state**

In the `<script>` of `frontend/src/routes/+page.svelte`, after `let pendingBadge = $state(0);`, add:

```svelte
	let myIds = $state<Set<string>>(new Set());
```

- [ ] **Step 2: Populate `myIds` from the existing own-rides fetch**

In `onMount`, inside the existing `if (phone) { try { ... } }` block, the line
`const mine = (await api.rides.list({}, phone)) as { ID: string }[];` already runs.
Right after that line, add:

```svelte
					myIds = new Set(mine.map((r) => r.ID));
```

(No extra request — reuse `mine`.)

- [ ] **Step 3: Pass `isOwn` to the Available-tab cards**

Change the Available-tab loop:

```svelte
						{#each rides as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} isOwn={myIds.has(r.ID)} />{/each}
```

- [ ] **Step 4: Verify the page still compiles/tests pass**

Run: `cd frontend && npx vitest run src/routes/home.test.ts`
Expected: PASS (no regression; existing home tests still green).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/+page.svelte
git commit -m "feat(home): mark own rides in the Available feed"
```

---

## Task 5: Search page wiring + test

**Files:**
- Modify: `frontend/src/routes/search/+page.svelte`
- Test: `frontend/src/routes/search.test.ts`

- [ ] **Step 1: Write the failing test**

Add this test inside the `describe('search', ...)` block in `frontend/src/routes/search.test.ts`. Also add the import for `userPhone` at the top of the file (alongside existing imports):

```ts
import { userPhone } from '$lib/stores';
```

```ts
	it('marks the user\'s own ride in results with a badge and manage link', async () => {
		userPhone.set('0612345678');
		const ownRide = {
			ID: '42', DriverName: 'Me', Origin: 'Saillans', Destination: 'Crest',
			DepartureAt: '2030-06-15T08:00:00Z', Flexibility: 0, InterestCount: 0
		};
		vi.stubGlobal('fetch', vi.fn(async (url: RequestInfo | URL) => {
			const u = String(url);
			if (u.includes('/api/rides')) return new Response(JSON.stringify([ownRide]), { status: 200 });
			return new Response('[]', { status: 200 }); // /api/interests, /api/destinations, etc.
		}));
		const { container } = render(Search);
		await vi.waitFor(() => {
			expect(container.querySelector('.tag-your-ride')).toBeTruthy();
			expect(container.querySelector('a.ride-manage-link')!.getAttribute('href')).toBe('/my-rides#card-42');
		});
		userPhone.set('');
	});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/routes/search.test.ts`
Expected: FAIL — no `.tag-your-ride` (search page doesn't compute own IDs yet).

- [ ] **Step 3: Wire own-ride IDs into the search page**

Edit `frontend/src/routes/search/+page.svelte`:

Add the import with the other imports:

```svelte
	import { loadMyRideIds } from '$lib/myRides';
```

Add state after `let contacts = $state<Map<string, string>>(new Map());`:

```svelte
	let myIds = $state<Set<string>>(new Set());
```

In `run()`, after the line `contacts = await loadAcceptedContacts(get(userPhone));`, add:

```svelte
		myIds = await loadMyRideIds(get(userPhone));
```

Update both result-column cards (the single `{#each col.list as r}` line):

```svelte
							{#each col.list as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} isOwn={myIds.has(r.ID)} />{/each}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/routes/search.test.ts`
Expected: PASS — the existing two-column test and the new own-ride test.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/routes/search/+page.svelte frontend/src/routes/search.test.ts
git commit -m "feat(search): mark the user's own rides in results"
```

---

## Task 6: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Full frontend test suite + type check**

Run: `cd frontend && npm test`
Expected: all suites pass.
Run (type check — if host `npm run check` fails with a `.svelte-kit` EACCES from the dev container, run it inside the container instead: `docker exec go-stop-frontend-1 sh -c "cd /app && npm run check"`).
Expected: `svelte-check` 0 errors, 0 warnings.

- [ ] **Step 2: Manual smoke (recommended)**

With the devstack running, set a profile (name+phone) and post a ride, then:
1. Home → Available tab: your posted ride shows a green "Your ride" badge and a "Manage in My rides →" link (no Request-contact button); other rides are unchanged.
2. `/search` for that route: same treatment in the result column; the link scrolls to the ride on `/my-rides`.
3. Clear the profile phone → reload → no rides are marked (normal browse).

---

## Self-Review Notes

- **Spec coverage:** badge + manage link replacing contact (Task 3) ✓; client-side ID match with no backend change (Tasks 2/4/5) ✓; home reuses existing fetch (Task 4) ✓; search uses helper (Task 5) ✓; i18n 2 keys × 7 locales (Task 1) ✓; empty-phone/error → empty set, nothing marked (Task 2 tests) ✓; RideCard + search tests (Tasks 3/5) ✓.
- **Type consistency:** `loadMyRideIds(phone): Promise<Set<string>>` defined in Task 2, used identically in Tasks 4/5; `isOwn` prop name consistent across Task 3 component and Tasks 4/5 call sites; `.tag-your-ride` and `.ride-manage-link` selectors consistent between Task 3 component and Tasks 3/5 tests.
- **Ordering:** i18n keys (Task 1) land before the component (Task 3) so `m.yourRide()`/`m.manageInMyRides()` exist at compile/test time.
