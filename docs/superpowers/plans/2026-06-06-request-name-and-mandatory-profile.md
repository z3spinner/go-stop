# Requester Name + Mandatory Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Always show the requester's name in the driver's view (incoming request and after accepting), and require a name + phone (held in the user's profile) before anyone can request a contact — enforced on both client and server.

**Architecture:** Frontend gate via a new `ProfileModal` that captures name+phone and resumes the original action; display changes in `MyRideCard`; server-side rejection of empty names in the express-interest path. No DB/schema changes — `searcher_name` already exists and is preserved client-side after accepting.

**Tech Stack:** Go (Gin) backend with usecase/handler layers; SvelteKit 2 + Svelte 5 runes frontend; Paraglide (inlang) i18n across 7 locales (base `fr`); Vitest + Testing Library; Go integration tests against Postgres.

---

## File Structure

- `internal/usecase/express_interest.go` — add empty-name guard (invariant, unit-tested).
- `internal/usecase/express_interest_test.go` — unit test for the guard.
- `internal/boundaries/handler/interest_handler.go` — return `400` on empty/whitespace name.
- `internal/boundaries/handler/integration_test.go` — new `RequiresName` test; update existing express-interest POSTs to include a name.
- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` — new i18n keys.
- `frontend/src/lib/profileModal.ts` — **new** store + opener (mirrors `notifModal.ts`).
- `frontend/src/lib/components/profile/ProfileModal.svelte` — **new** modal component.
- `frontend/src/lib/components/profile/ProfileModal.test.ts` — **new** test.
- `frontend/src/routes/+layout.svelte` — mount `<ProfileModal />`.
- `frontend/src/lib/components/rides/ContactOrInterest.svelte` — gate the request.
- `frontend/src/lib/components/rides/ContactOrInterest.test.ts` — gating test.
- `frontend/src/lib/components/rides/MyRideCard.svelte` — labeled name display.
- `frontend/src/lib/components/rides/MyRideCard.test.ts` — **new** display test.

---

## Task 1: Backend — usecase rejects empty searcher name

**Files:**
- Modify: `internal/usecase/express_interest.go`
- Test: `internal/usecase/express_interest_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to the end of `internal/usecase/express_interest_test.go`:

```go
func TestExpressInterest_RejectsEmptyName(t *testing.T) {
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver"},
		},
	}
	interests := &mockInterestRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	_, err := uc.Execute("ride-1", "555-searcher", "")

	if !errors.Is(err, usecase.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}
	if len(interests.saved) != 0 {
		t.Error("no interest should be saved when name is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/usecase/ -run TestExpressInterest_RejectsEmptyName -v`
Expected: FAIL — `usecase.ErrNameRequired` undefined (compile error).

- [ ] **Step 3: Add the sentinel error and guard**

In `internal/usecase/express_interest.go`, add the error near the top (after the imports, before the struct):

```go
var ErrNameRequired = errors.New("name is required")
```

Then in `Execute`, after the driver check (`if ride.Phone == searcherPhone { ... }`) and before building the `interest`, add:

```go
	if strings.TrimSpace(searcherName) == "" {
		return domain.Interest{}, ErrNameRequired
	}
```

Add `"strings"` to the import block.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/usecase/ -run TestExpressInterest -v`
Expected: PASS (all `TestExpressInterest_*`, including the existing ones — `ReturnsErrorIfRideNotFound` still errors on the ride lookup before the name check).

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/express_interest.go internal/usecase/express_interest_test.go
git commit -m "feat(interest): reject empty searcher name in express usecase"
```

---

## Task 2: Backend — handler returns 400 on empty name

**Files:**
- Modify: `internal/boundaries/handler/interest_handler.go:39-81`
- Test: `internal/boundaries/handler/integration_test.go`

- [ ] **Step 1: Update existing integration tests to send a name**

These existing express-interest POSTs must include a `name` (otherwise they'll now 400). Apply each edit:

`internal/boundaries/handler/integration_test.go:403-405` (TestHTTP_Interest_ExpressCreatesRecord):

```go
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002", "name": "Bob",
	})
```

`:435-437` (TestHTTP_Interest_DriverCannotBeSearcher) — keep the driver phone, add a name so it reaches the 403 driver check:

```go
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550001", "name": "Alice",
	})
```

`:458-460` (TestHTTP_Interest_AcceptRevealsPhonesCorrectly):

```go
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002", "name": "Bob",
	})
```

`:563-565` (wrong-driver accept test):

```go
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002", "name": "Bob",
	})
```

(The POST at `:1044` already includes `"name": "Bob"` — leave it.)

- [ ] **Step 2: Write the failing test**

Add this test to `internal/boundaries/handler/integration_test.go` (next to the other `TestHTTP_Interest_*` tests):

```go
func TestHTTP_Interest_RequiresName(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "5550001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	// whitespace-only name is rejected
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "5550002", "name": "   ",
	})
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d: %s", w2.Code, w2.Body.String())
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `make test` (requires Postgres — start it with `docker compose up -d db` if needed; the Makefile sets `TEST_DATABASE_URL`).
Expected: FAIL — `TestHTTP_Interest_RequiresName` gets 201 instead of 400 (whitespace name currently accepted).

- [ ] **Step 4: Add the 400 guard in the handler**

In `internal/boundaries/handler/interest_handler.go`, inside `Express`, after `ShouldBindJSON` succeeds and before calling the usecase, add the trimmed-name check:

```go
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	interest, err := h.expressInterest.Execute(c.Param("id"), normalizePhone(req.Phone), name)
```

(Replace the existing `interest, err := h.expressInterest.Execute(...)` line. `strings` is already imported.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `make test`
Expected: PASS — `TestHTTP_Interest_RequiresName` returns 400; all updated `TestHTTP_Interest_*` tests still pass (403 for driver, 201 for valid-name expresses).

- [ ] **Step 6: Commit**

```bash
git add internal/boundaries/handler/interest_handler.go internal/boundaries/handler/integration_test.go
git commit -m "feat(interest): require name on express-interest endpoint (400 if missing)"
```

---

## Task 3: Frontend — i18n message keys (all 7 locales)

**Files:**
- Modify: `frontend/src/messages/fr.json`, `en.json`, `es.json`, `it.json`, `de.json`, `nl.json`, `el.json`

No test step (data only); verified by Task 6's render test and the paraglide compile.

- [ ] **Step 1: Add the keys to every locale file**

Add these four keys to each top-level JSON object (place them next to the existing interest keys, e.g. right after `"contactRevealed"`; ensure commas remain valid).

`fr.json` (base):
```json
	"interestPendingName": "{name} demande un trajet",
	"anonymousSearcher": "Quelqu'un",
	"profileRequiredTitle": "Complétez votre profil",
	"profileRequiredBody": "Ajoutez votre nom et votre numéro pour que le conducteur sache qui demande.",
	"btnSaveContinue": "Enregistrer et continuer",
```

`en.json`:
```json
	"interestPendingName": "{name} wants a ride",
	"anonymousSearcher": "Someone",
	"profileRequiredTitle": "Complete your profile",
	"profileRequiredBody": "Add your name and phone number so the driver knows who's asking.",
	"btnSaveContinue": "Save & continue",
```

`es.json`:
```json
	"interestPendingName": "{name} busca un viaje",
	"anonymousSearcher": "Alguien",
	"profileRequiredTitle": "Completa tu perfil",
	"profileRequiredBody": "Añade tu nombre y número para que el conductor sepa quién lo pide.",
	"btnSaveContinue": "Guardar y continuar",
```

`it.json`:
```json
	"interestPendingName": "{name} cerca un passaggio",
	"anonymousSearcher": "Qualcuno",
	"profileRequiredTitle": "Completa il tuo profilo",
	"profileRequiredBody": "Aggiungi nome e numero così il conducente sa chi lo chiede.",
	"btnSaveContinue": "Salva e continua",
```

`de.json`:
```json
	"interestPendingName": "{name} sucht eine Mitfahrt",
	"anonymousSearcher": "Jemand",
	"profileRequiredTitle": "Profil vervollständigen",
	"profileRequiredBody": "Gib deinen Namen und deine Nummer an, damit der Fahrer weiß, wer fragt.",
	"btnSaveContinue": "Speichern & weiter",
```

`nl.json`:
```json
	"interestPendingName": "{name} zoekt een rit",
	"anonymousSearcher": "Iemand",
	"profileRequiredTitle": "Vul je profiel in",
	"profileRequiredBody": "Voeg je naam en nummer toe zodat de bestuurder weet wie het vraagt.",
	"btnSaveContinue": "Opslaan en doorgaan",
```

`el.json`:
```json
	"interestPendingName": "Ο/Η {name} ζητάει διαδρομή",
	"anonymousSearcher": "Κάποιος",
	"profileRequiredTitle": "Συμπληρώστε το προφίλ σας",
	"profileRequiredBody": "Προσθέστε όνομα και τηλέφωνο ώστε ο οδηγός να ξέρει ποιος ρωτάει.",
	"btnSaveContinue": "Αποθήκευση & συνέχεια",
```

- [ ] **Step 2: Compile paraglide and verify no errors**

Run: `cd frontend && npm run paraglide`
Expected: completes with no missing-key/parse errors; the new message functions are generated under `src/lib/paraglide/messages`.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/messages
git commit -m "i18n: add keys for requester name + profile-required modal"
```

---

## Task 4: Frontend — ProfileModal store + component

**Files:**
- Create: `frontend/src/lib/profileModal.ts`
- Create: `frontend/src/lib/components/profile/ProfileModal.svelte`
- Create: `frontend/src/lib/components/profile/ProfileModal.test.ts`
- Modify: `frontend/src/routes/+layout.svelte`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/lib/components/profile/ProfileModal.test.ts`:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import ProfileModal from './ProfileModal.svelte';
import { profileModalState, openProfileModal } from '$lib/profileModal';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => {
	localStorage.clear();
	userName.set('');
	userPhone.set('');
	profileModalState.set(null);
});

describe('ProfileModal', () => {
	it('saves name + phone and runs the continuation, then closes', async () => {
		const onComplete = vi.fn();
		render(ProfileModal);
		openProfileModal(onComplete);

		const name = await screen.findByRole('textbox', { name: /name|nom/i });
		await fireEvent.input(name, { target: { value: 'Marie' } });
		const phone = document.querySelector('input[type="tel"]') as HTMLInputElement;
		await fireEvent.input(phone, { target: { value: '0612345678' } });

		await fireEvent.click(screen.getByRole('button', { name: /continue|continuer/i }));

		await vi.waitFor(() => {
			expect(onComplete).toHaveBeenCalledTimes(1);
			expect(get(userName)).toBe('Marie');
			expect(get(userPhone)).toBe('0612345678');
			expect(get(profileModalState)).toBeNull();
		});
	});

	it('disables the save button until both fields are filled', async () => {
		render(ProfileModal);
		openProfileModal(vi.fn());
		const btn = (await screen.findByRole('button', { name: /continue|continuer/i })) as HTMLButtonElement;
		expect(btn.disabled).toBe(true);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/components/profile/ProfileModal.test.ts`
Expected: FAIL — cannot resolve `$lib/profileModal` / `./ProfileModal.svelte` (not created yet).

- [ ] **Step 3: Create the store**

Create `frontend/src/lib/profileModal.ts`:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { writable } from 'svelte/store';

// Holds the action to run once the profile is completed; null when closed.
export const profileModalState = writable<(() => void) | null>(null);

export function openProfileModal(onComplete: () => void) {
	profileModalState.set(onComplete);
}
```

- [ ] **Step 4: Create the component**

Create `frontend/src/lib/components/profile/ProfileModal.svelte`:

```svelte
<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import * as Dialog from '$lib/components/ui/dialog';
	import { profileModalState } from '$lib/profileModal';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';

	let onComplete = $derived($profileModalState);
	let open = $derived(onComplete !== null);

	let name = $state(get(userName));
	let phone = $state(get(userPhone));

	const complete = $derived(name.trim().length > 0 && phone.trim().length > 0);

	function close() {
		profileModalState.set(null);
	}
	function onOpenChange(v: boolean) {
		if (!v) close();
	}

	function saveAndContinue() {
		if (!complete) return;
		userName.set(name.trim());
		userPhone.set(normalizePhone(phone));
		const cb = onComplete;
		close();
		cb?.();
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.profileRequiredTitle()}</Dialog.Title>
		</Dialog.Header>
		<p class="text-sm text-gray-600">{m.profileRequiredBody()}</p>
		<div class="mt-1 flex flex-col gap-2">
			<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={name} /></label>
			<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={phone} /></label>
		</div>
		<div class="mt-3 flex gap-2">
			<button type="button" id="btn-profile-save" class="btn btn-primary" disabled={!complete} onclick={saveAndContinue}>{m.btnSaveContinue()}</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/lib/components/profile/ProfileModal.test.ts`
Expected: PASS (both tests).

- [ ] **Step 6: Mount the modal in the layout**

In `frontend/src/routes/+layout.svelte`, add the import alongside the other component imports (near the `NotifModal` import, ~line 24):

```svelte
	import ProfileModal from '$lib/components/profile/ProfileModal.svelte';
```

And add the element next to `<NotifModal />` in the markup:

```svelte
<NotifModal />
<ProfileModal />
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/lib/profileModal.ts frontend/src/lib/components/profile/ frontend/src/routes/+layout.svelte
git commit -m "feat(profile): add ProfileModal to capture name + phone before gated actions"
```

---

## Task 5: Frontend — gate ContactOrInterest behind a complete profile

**Files:**
- Modify: `frontend/src/lib/components/rides/ContactOrInterest.svelte:23-41`
- Test: `frontend/src/lib/components/rides/ContactOrInterest.test.ts`

- [ ] **Step 1: Write the failing test**

Add this test inside the `describe('ContactOrInterest', ...)` block in `frontend/src/lib/components/rides/ContactOrInterest.test.ts`:

```ts
	it('opens the profile modal instead of sending when the profile is incomplete', async () => {
		userName.set(''); // incomplete: no name
		userPhone.set('0622000002');
		const fetchMock = vi.fn(async () => new Response(JSON.stringify({ id: 'int1', status: 'pending' }), { status: 201 }));
		vi.stubGlobal('fetch', fetchMock);

		const { container } = render(ContactOrInterest, { props: { ride } });
		await fireEvent.click(container.querySelector('.btn-interest')!);

		const { get } = await import('svelte/store');
		const { profileModalState } = await import('$lib/profileModal');
		expect(get(profileModalState)).toBeTypeOf('function'); // modal opened
		expect(fetchMock).not.toHaveBeenCalled(); // nothing sent yet
	});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/components/rides/ContactOrInterest.test.ts`
Expected: FAIL — current code calls `window.prompt`/fetch instead of opening the modal; `profileModalState` stays null.

- [ ] **Step 3: Update `express()` to gate on a complete profile**

In `frontend/src/lib/components/rides/ContactOrInterest.svelte`, add the import (with the other imports):

```svelte
	import { openProfileModal } from '$lib/profileModal';
	import { normalizePhone } from '$lib/utils';
```

Replace the `express()` function (lines ~23-41) with:

```svelte
	async function express() {
		if (busy) return;
		const name = get(userName).trim();
		const phone = get(userPhone).trim();
		if (!name || !phone) {
			openProfileModal(() => express()); // resume once the profile is complete
			return;
		}
		busy = true;
		stateMsg = '';
		try {
			const res = await api.interests.express(ride.ID, normalizePhone(phone), name);
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
```

This removes the phone-only `window.prompt` fallback and always sends a non-empty name.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/lib/components/rides/ContactOrInterest.test.ts`
Expected: PASS — the new gating test passes, and the existing "expressing interest POSTs…" test still passes (its `beforeEach` sets `userName='Bob'`, so the profile is complete).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/components/rides/ContactOrInterest.svelte frontend/src/lib/components/rides/ContactOrInterest.test.ts
git commit -m "feat(interest): require complete profile before requesting contact"
```

---

## Task 6: Frontend — show labeled name in MyRideCard

**Files:**
- Modify: `frontend/src/lib/components/rides/MyRideCard.svelte:61-75`
- Test: `frontend/src/lib/components/rides/MyRideCard.test.ts` (new)

- [ ] **Step 1: Write the failing test**

Create `frontend/src/lib/components/rides/MyRideCard.test.ts`:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';

const listInterests = vi.fn();
vi.mock('$lib/api', () => ({
	api: {
		rides: {
			listMatchingRequests: vi.fn(async () => []),
			listInterests,
		},
	},
}));

import MyRideCard from './MyRideCard.svelte';

const ride = {
	ID: 'r1', Origin: 'Saillans', Destination: 'Crest',
	DepartureAt: '2030-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;

beforeEach(() => listInterests.mockReset());

describe('MyRideCard name display', () => {
	it('shows "{name} wants a ride" for a pending interest', async () => {
		listInterests.mockResolvedValue([{ id: 'i1', status: 'pending', searcher_name: 'Marie' }]);
		render(MyRideCard, { props: { ride, phone: '5550001' } });
		expect(await screen.findByText(/Marie/)).toBeTruthy();
		expect(screen.getByText(/wants a ride|demande un trajet/)).toBeTruthy();
	});

	it('shows name and phone for an accepted interest', async () => {
		listInterests.mockResolvedValue([
			{ id: 'i2', status: 'accepted', searcher_name: 'Marie', searcher_phone: '0612345678' },
		]);
		const { container } = render(MyRideCard, { props: { ride, phone: '5550001' } });
		await screen.findByText(/Marie/);
		expect(container.querySelector('a[href="tel:0612345678"]')).toBeTruthy();
	});

	it('falls back to a placeholder when a pending interest has no name', async () => {
		listInterests.mockResolvedValue([{ id: 'i3', status: 'pending', searcher_name: '' }]);
		render(MyRideCard, { props: { ride, phone: '5550001' } });
		expect(await screen.findByText(/Someone|Quelqu'un/)).toBeTruthy();
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/components/rides/MyRideCard.test.ts`
Expected: FAIL — accepted row shows "Contact accepted" not the name; placeholder not rendered.

- [ ] **Step 3: Update the interests markup**

In `frontend/src/lib/components/rides/MyRideCard.svelte`, add a display-name helper inside the `<script>` block (e.g. after the `accept` function):

```svelte
	const displayName = (it: InterestListItem) => it.searcher_name?.trim() || m.anonymousSearcher();
```

Then replace the three interest-row branches (lines ~65-72) with:

```svelte
				{#if it.status === 'pending'}
					<span class="interest-pending-info">{m.interestPendingName({ name: displayName(it) })}</span>
					<button type="button" class="btn-accept-interest" data-id={it.id} data-phone={phone} onclick={() => accept(it)}>{m.btnAccept()}</button>
				{:else if it.status === 'driver_shared'}
					<span class="interest-accepted">{m.notifSentShort()}</span>
				{:else}
					<span class="interest-accepted">{displayName(it)}{#if it.searcher_phone} — <a href="tel:{it.searcher_phone}">{it.searcher_phone}</a>{/if}</span>
				{/if}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/lib/components/rides/MyRideCard.test.ts`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/components/rides/MyRideCard.svelte frontend/src/lib/components/rides/MyRideCard.test.ts
git commit -m "feat(my-rides): show requester name on pending and accepted interests"
```

---

## Task 7: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full frontend test + type check**

Run: `cd frontend && npm test && npm run check`
Expected: all Vitest suites pass; `svelte-check` reports 0 errors.

- [ ] **Step 2: Run the full backend test suite**

Run: `make test` (Postgres must be available; `docker compose up -d db` if needed)
Expected: all Go tests pass.

- [ ] **Step 3: Run the linter**

Run: `make lint` (and `cd frontend && npm run check` already covers the frontend)
Expected: no new lint findings.

- [ ] **Step 4: Manual smoke (optional, recommended)**

With `docker compose up`, in the browser:
1. Clear your profile (localStorage) → open a ride → "Request contact" → the ProfileModal appears; the request is **not** sent until you fill name + phone and click "Save & continue".
2. As the driver (`/my-rides`), confirm the pending row reads "{name} wants a ride", and after Accept it reads "{name} — {phone}" with a working `tel:` link.

---

## Self-Review Notes

- **Spec coverage:** Display (Task 6) ✓; gate via Profile modal (Tasks 4–5) ✓; server-side enforcement (Tasks 1–2) ✓; i18n across 7 locales (Task 3) ✓; placeholder for legacy empty names (Task 6) ✓; notification-text fix is a consequence of Tasks 1–2 ✓.
- **Type consistency:** `openProfileModal(onComplete)` / `profileModalState` used identically in Tasks 4 and 5; `displayName(it)` defined and used within Task 6; `ErrNameRequired` defined in Task 1 and relied on by the handler 400 in Task 2.
- **Existing-test impact:** Task 2 Step 1 updates the four name-less express POSTs so the new 400 rule doesn't break them; Task 5 relies on `ContactOrInterest.test.ts`'s existing `beforeEach` profile so the legacy express test still passes.
