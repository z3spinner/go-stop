# Repeat-Ride Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a driver repeat a posted ride daily / on weekdays / weekly for a chosen number of times, expanding into N independent ride posts client-side.

**Architecture:** A pure date-math helper computes whole-day offsets per occurrence; `RideForm` gains a Repeat control and loops the existing idempotent `api.rides.post` once per occurrence (and once per return leg). No backend or schema change.

**Tech Stack:** SvelteKit (Svelte 5 runes), Paraglide i18n, Vitest + @testing-library/svelte.

---

## File structure

**Create**
- `frontend/src/lib/recurrence.ts` — pure date math: `expandOffsets`, `shiftDaysIso`, `Frequency` type.
- `frontend/src/lib/recurrence.test.ts` — unit tests for the helper.
- `frontend/src/lib/components/rides/ride-form.test.ts` — RideForm component test.

**Modify**
- `frontend/src/lib/components/rides/RideForm.svelte` — Repeat control + expansion loop.
- `frontend/src/messages/{en,fr,es,it,de,nl,el}.json` — repeat UI message keys.

Commands run from `frontend/`.

---

## Task 1: Recurrence date-math helper

**Files:**
- Create: `frontend/src/lib/recurrence.ts`
- Test: `frontend/src/lib/recurrence.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/lib/recurrence.test.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { expandOffsets, shiftDaysIso } from './recurrence';

// 2024-01-01 is a Monday; 2024-01-06 is a Saturday (stable real dates).
const MON = new Date(2024, 0, 1);
const SAT = new Date(2024, 0, 6);

describe('expandOffsets', () => {
	it('returns [0] for none, ignoring count', () => {
		expect(expandOffsets(MON, 'none', 5)).toEqual([0]);
	});

	it('daily is consecutive days', () => {
		expect(expandOffsets(MON, 'daily', 3)).toEqual([0, 1, 2]);
	});

	it('weekly steps by 7 days', () => {
		expect(expandOffsets(MON, 'weekly', 3)).toEqual([0, 7, 14]);
	});

	it('count is the total, so count 1 yields a single occurrence', () => {
		expect(expandOffsets(MON, 'daily', 1)).toEqual([0]);
	});

	it('weekdays skips weekends from a weekday base', () => {
		// Mon..Fri = 0..4, skip Sat(5)/Sun(6), next Mon = 7
		expect(expandOffsets(MON, 'weekdays', 6)).toEqual([0, 1, 2, 3, 4, 7]);
	});

	it('weekdays from a weekend base starts at the next weekday', () => {
		// Sat base: skip Sat(0)/Sun(1), first weekday Mon = offset 2
		expect(expandOffsets(SAT, 'weekdays', 1)).toEqual([2]);
	});
});

describe('shiftDaysIso', () => {
	it('preserves local time and date at offset 0', () => {
		const d = new Date(shiftDaysIso('2030-06-03T08:00', 0));
		expect(d.getMonth()).toBe(5);
		expect(d.getDate()).toBe(3);
		expect(d.getHours()).toBe(8);
		expect(d.getMinutes()).toBe(0);
	});

	it('shifts the local date by N days, preserving time of day', () => {
		const d = new Date(shiftDaysIso('2030-06-03T08:00', 5));
		expect(d.getDate()).toBe(8);
		expect(d.getHours()).toBe(8);
	});

	it('rolls over month boundaries', () => {
		const d = new Date(shiftDaysIso('2030-06-30T08:00', 2));
		expect(d.getMonth()).toBe(6); // July
		expect(d.getDate()).toBe(2);
	});
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run src/lib/recurrence.test.ts`
Expected: FAIL — cannot resolve `./recurrence`.

- [ ] **Step 3: Implement the helper**

Create `frontend/src/lib/recurrence.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

export type Frequency = 'none' | 'daily' | 'weekdays' | 'weekly';

/**
 * Whole-day offsets from the base date for each occurrence.
 * - none      → [0] (count ignored)
 * - daily(n)  → [0, 1, …, n-1]
 * - weekly(n) → [0, 7, …, 7(n-1)]
 * - weekdays  → base stepped day-by-day counting only Mon–Fri until n collected;
 *               a weekend base is skipped, so offsets[0] may be > 0.
 */
export function expandOffsets(base: Date, frequency: Frequency, count: number): number[] {
	if (frequency === 'none') return [0];
	const n = Math.max(1, Math.floor(count));
	if (frequency === 'daily') return Array.from({ length: n }, (_, i) => i);
	if (frequency === 'weekly') return Array.from({ length: n }, (_, i) => i * 7);
	// weekdays
	const offsets: number[] = [];
	for (let d = 0; offsets.length < n; d++) {
		const day = new Date(base.getFullYear(), base.getMonth(), base.getDate() + d).getDay();
		if (day !== 0 && day !== 6) offsets.push(d);
	}
	return offsets;
}

/**
 * Shift a `datetime-local` string ("YYYY-MM-DDTHH:MM") by whole days, preserving
 * the local time of day, and return an ISO (UTC) string. Day-stepping via local
 * date components keeps the wall-clock time stable across DST changes.
 */
export function shiftDaysIso(localDateTime: string, days: number): string {
	const d = new Date(localDateTime);
	return new Date(
		d.getFullYear(),
		d.getMonth(),
		d.getDate() + days,
		d.getHours(),
		d.getMinutes()
	).toISOString();
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `npx vitest run src/lib/recurrence.test.ts`
Expected: PASS (9 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/recurrence.ts frontend/src/lib/recurrence.test.ts
git commit -m "feat(rides): add recurrence date-math helper"
```

---

## Task 2: Repeat UI message keys

**Files:**
- Modify: `frontend/src/messages/{en,fr,es,it,de,nl,el}.json`

- [ ] **Step 1: Add the keys to every locale file**

Each file is a flat JSON object. Append these key/value pairs (insert before the
closing `}`, keeping valid JSON — add a comma after the previous last entry).
`{count}`, `{first}`, `{last}` are inlang placeholders; keep them literally.

`en.json`:
```json
"repeatLabel": "Repeat",
"repeatNone": "Don't repeat",
"repeatDaily": "Daily",
"repeatWeekdays": "Weekdays",
"repeatWeekly": "Weekly",
"repeatCountLabel": "Number of rides",
"repeatSummary": "Creates {count} rides · {first} → {last}"
```

`fr.json`:
```json
"repeatLabel": "Répéter",
"repeatNone": "Ne pas répéter",
"repeatDaily": "Chaque jour",
"repeatWeekdays": "En semaine",
"repeatWeekly": "Chaque semaine",
"repeatCountLabel": "Nombre de trajets",
"repeatSummary": "Crée {count} trajets · {first} → {last}"
```

`es.json`:
```json
"repeatLabel": "Repetir",
"repeatNone": "No repetir",
"repeatDaily": "Cada día",
"repeatWeekdays": "Días laborables",
"repeatWeekly": "Cada semana",
"repeatCountLabel": "Número de viajes",
"repeatSummary": "Crea {count} viajes · {first} → {last}"
```

`it.json`:
```json
"repeatLabel": "Ripeti",
"repeatNone": "Non ripetere",
"repeatDaily": "Ogni giorno",
"repeatWeekdays": "Giorni feriali",
"repeatWeekly": "Ogni settimana",
"repeatCountLabel": "Numero di viaggi",
"repeatSummary": "Crea {count} viaggi · {first} → {last}"
```

`de.json`:
```json
"repeatLabel": "Wiederholen",
"repeatNone": "Nicht wiederholen",
"repeatDaily": "Täglich",
"repeatWeekdays": "Wochentags",
"repeatWeekly": "Wöchentlich",
"repeatCountLabel": "Anzahl der Fahrten",
"repeatSummary": "Erstellt {count} Fahrten · {first} → {last}"
```

`nl.json`:
```json
"repeatLabel": "Herhalen",
"repeatNone": "Niet herhalen",
"repeatDaily": "Dagelijks",
"repeatWeekdays": "Doordeweeks",
"repeatWeekly": "Wekelijks",
"repeatCountLabel": "Aantal ritten",
"repeatSummary": "Maakt {count} ritten · {first} → {last}"
```

`el.json`:
```json
"repeatLabel": "Επανάληψη",
"repeatNone": "Χωρίς επανάληψη",
"repeatDaily": "Καθημερινά",
"repeatWeekdays": "Καθημερινές",
"repeatWeekly": "Εβδομαδιαία",
"repeatCountLabel": "Αριθμός διαδρομών",
"repeatSummary": "Δημιουργεί {count} διαδρομές · {first} → {last}"
```

- [ ] **Step 2: Recompile and validate**

Run: `npm run paraglide`
Expected: "Successfully compiled inlang project." If a JSON comma is wrong, the
compile fails — fix and re-run.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/messages frontend/src/lib/paraglide
git commit -m "feat(rides): add repeat-ride message keys (7 locales)"
```

(Note: `frontend/src/lib/paraglide/` is gitignored; the `git add` is a no-op for
it, which is expected.)

---

## Task 3: Wire the Repeat control into RideForm

**Files:**
- Modify: `frontend/src/lib/components/rides/RideForm.svelte`
- Test: `frontend/src/lib/components/rides/ride-form.test.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/lib/components/rides/ride-form.test.ts`:
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

const post = vi.fn(async () => ({}));
vi.mock('$lib/api', () => ({ api: { rides: { post: (b: unknown) => post(b) } } }));

import RideForm from './RideForm.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => {
	post.mockClear();
	userName.set('Marie');
	userPhone.set('0612345678');
});

async function fillBase(container: HTMLElement) {
	await fireEvent.input(container.querySelector('input[name=origin]')!, { target: { value: 'Saillans' } });
	await fireEvent.input(container.querySelector('input[name=destination]')!, { target: { value: 'Crest' } });
	await fireEvent.input(container.querySelector('input[name=departure_at]')!, { target: { value: '2030-06-03T08:00' } });
}

describe('RideForm repeat', () => {
	it('posts one ride when not repeating (regression)', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.submit(container.querySelector('#ride-form')!);
		expect(post).toHaveBeenCalledTimes(1);
	});

	it('daily × 3 posts three rides on consecutive days at the same time', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.change(container.querySelector('#repeat-frequency')!, { target: { value: 'daily' } });
		await fireEvent.input(container.querySelector('#repeat-count')!, { target: { value: '3' } });
		await fireEvent.submit(container.querySelector('#ride-form')!);

		expect(post).toHaveBeenCalledTimes(3);
		const days = post.mock.calls.map((c) => new Date((c[0] as { departure_at: string }).departure_at).getDate());
		expect(days).toEqual([3, 4, 5]);
		const hours = post.mock.calls.map((c) => new Date((c[0] as { departure_at: string }).departure_at).getHours());
		expect(hours).toEqual([8, 8, 8]);
	});

	it('repeat with return posts both legs per occurrence', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.click(container.querySelector('#btn-return')!);
		await fireEvent.change(container.querySelector('#repeat-frequency')!, { target: { value: 'daily' } });
		await fireEvent.input(container.querySelector('#repeat-count')!, { target: { value: '2' } });
		await fireEvent.submit(container.querySelector('#ride-form')!);

		// 2 outbound + 2 return
		expect(post).toHaveBeenCalledTimes(4);
		const returns = post.mock.calls.filter((c) => (c[0] as { origin: string }).origin === 'Crest');
		expect(returns).toHaveLength(2);
	});
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `npx vitest run src/lib/components/rides/ride-form.test.ts`
Expected: FAIL — `#repeat-frequency` not found / only 1 post for the daily case.

- [ ] **Step 3: Add recurrence state and imports**

In `frontend/src/lib/components/rides/RideForm.svelte`, the script currently ends
its imports with `import type { Flexibility } from '$lib/types';`. Add directly
below that import:
```ts
	import { expandOffsets, shiftDaysIso, type Frequency } from '$lib/recurrence';
	import { formatDate } from '$lib/utils';
```

Then, directly below the existing `let return_flexibility = $state<Flexibility>(30);`
line, add:
```ts
	let frequency = $state<Frequency>('none');
	let repeatCount = $state(4);
	let count = $derived(Math.min(14, Math.max(1, Math.floor(Number(repeatCount) || 1))));
	let offsets = $derived(expandOffsets(new Date(departure_at), frequency, frequency === 'none' ? 1 : count));
	let summary = $derived.by(() => {
		if (frequency === 'none' || offsets.length === 0) return '';
		const first = formatDate(shiftDaysIso(departure_at, offsets[0]));
		const last = formatDate(shiftDaysIso(departure_at, offsets[offsets.length - 1]));
		return m.repeatSummary({ count: offsets.length, first, last });
	});
```

- [ ] **Step 4: Replace the submit post block with the expansion loop**

In the same file, replace this exact block:
```ts
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
```
with:
```ts
		try {
			for (const off of offsets) {
				await api.rides.post({
					driver_name, phone: ph, origin, destination,
					departure_at: shiftDaysIso(departure_at, off), flexibility
				});
				if (isReturn && return_departure_at) {
					await api.rides.post({
						driver_name, phone: ph, origin: destination, destination: origin,
						departure_at: shiftDaysIso(return_departure_at, off), flexibility: return_flexibility
					});
				}
			}
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
```

- [ ] **Step 5: Add the Repeat control to the template**

In the same file, find the submit button line:
```svelte
	<button type="submit" class="btn btn-primary">{m.btnPostRide()}</button>
```
Insert directly **above** it:
```svelte
	<label for="repeat-frequency">{m.repeatLabel()}
		<select id="repeat-frequency" bind:value={frequency}>
			<option value="none">{m.repeatNone()}</option>
			<option value="daily">{m.repeatDaily()}</option>
			<option value="weekdays">{m.repeatWeekdays()}</option>
			<option value="weekly">{m.repeatWeekly()}</option>
		</select>
	</label>
	{#if frequency !== 'none'}
		<label for="repeat-count">{m.repeatCountLabel()}
			<input id="repeat-count" type="number" min="1" max="14" bind:value={repeatCount} />
		</label>
		{#if summary}<p class="text-sm text-gray-600">{summary}</p>{/if}
	{/if}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `npx vitest run src/lib/components/rides/ride-form.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 7: Commit**

```bash
git add frontend/src/lib/components/rides/RideForm.svelte frontend/src/lib/components/rides/ride-form.test.ts
git commit -m "feat(rides): repeat a ride daily/weekdays/weekly on post"
```

---

## Task 4: Verification (and devstack hand-off)

**Files:** none (verification only)

- [ ] **Step 1: Full unit suite**

Run: `npx vitest run`
Expected: all suites PASS (existing + the new recurrence and ride-form tests).

- [ ] **Step 2: Type-check (in Docker — host `.svelte-kit` is root-owned)**

Run from repo root:
```bash
docker compose run --rm --no-deps frontend sh -c "npm run check"
```
Expected: `svelte-check found 0 errors and 0 warnings`.

- [ ] **Step 3: Production build**

Run from repo root:
```bash
docker compose run --rm --no-deps frontend sh -c "npm run build"
```
Expected: build succeeds (`✔ done`).

- [ ] **Step 4: Manual devstack test (user gate — do NOT push before this)**

The dev stack runs on http://localhost:5173 (hot-reloads from `frontend/`).
The user verifies in the browser:
- Post-ride → choose **Daily**, count 3 → submit → **My rides** shows 3 rides on
  consecutive days at the same time.
- **Weekdays** spanning a weekend → no Saturday/Sunday rides.
- **Weekly** count 3 → three rides 7 days apart.
- **Return trip + repeat** → each day has an outbound and a return.
- **Don't repeat** → exactly one ride (unchanged behaviour).

Only after the user confirms in the devstack should the branch be merged/pushed.

---

## Self-review notes

- **Spec coverage:** client-side expansion (T3 loop), frequencies + weekday skip
  (T1 `expandOffsets`), count = total + cap 14 (T3 `count` derived), return per
  occurrence (T3 loop), time-of-day preserved across days (T1 `shiftDaysIso`),
  Repeat select + count + summary (T3 template), i18n keys (T2), sequential posts
  with error surfaced (T3 try/catch), tests (T1 unit + T3 component). All covered.
- **Type consistency:** `Frequency` from `recurrence.ts` used in `RideForm`;
  `expandOffsets(base: Date, frequency, count)` and `shiftDaysIso(string, number)`
  signatures match every call site.
- No backend changes (reuses idempotent `api.rides.post`).
