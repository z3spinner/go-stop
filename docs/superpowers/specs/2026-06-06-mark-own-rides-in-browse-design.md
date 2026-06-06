# Mark the user's own rides in browse / search

**Date:** 2026-06-06
**Status:** Approved (design)

## Problem

A user is sometimes a driver too. When they browse the "Available" rides on the
home page or search for a ride on `/search`, their own published rides appear in
the results with no indication that they're theirs — and each result card offers a
"Request contact" button that, on your own ride, does nothing useful (the server
rejects a driver expressing interest in their own ride with `403`).

## Goal

When the user's own rides appear in a browse/search list, make it clearly visible
that they're the user's own, and replace the now-pointless contact action with a
link to manage the ride.

## Non-goals

- No filtering: own rides still appear in the list (explicit user request) — just
  marked.
- No accounts/auth change — identity remains the phone number in `localStorage`.
- No change to the privacy model: the driver's phone stays **out** of `PublicRide`.
- No change to `/my-rides` itself.

## Approach — client-side ID match, no backend change

Public ride objects (`PublicRide`) deliberately omit the driver's phone, so "is
this mine?" cannot be matched on phone. Instead we match on **ride ID**: the
browse/search pages know the current user's phone (`userPhone` store) and can fetch
the set of the user's own ride IDs, then mark any card whose `ride.ID` is in that
set.

This keeps the privacy model intact and avoids overloading the search/list endpoint
(which today returns the full owned `Ride[]` — with phone — when an `X-Phone`
header is present, and the public feed otherwise).

## Design

### 1. `RideCard.svelte` — the shared browse/search card

`frontend/src/lib/components/rides/RideCard.svelte` is rendered by both the home
"Available" tab and both `/search` result columns. Add an `isOwn` prop:

```ts
let { ride, contactPhone, showDriver = true, isOwn = false }:
  { ride: PublicRide | Ride; contactPhone?: string; showDriver?: boolean; isOwn?: boolean } = $props();
```

When `isOwn` is true:

- **Badge** in the meta row (`.card-meta`), placed first so it reads clearly,
  styled like the existing `.tag` but green to stand apart from the blue
  `.tag-interest-count`:

  ```svelte
  {#if isOwn}<span class="tag tag-your-ride rounded bg-green-100 px-1">{m.yourRide()}</span>{/if}
  ```

- **Action area**: replace the `<ContactOrInterest>` block with a manage link to
  the user's own ride on `/my-rides`. `MyRideCard.svelte` already renders
  `id="card-{ride.ID}"`, so the hash scrolls to that specific ride:

  ```svelte
  {#if isOwn}
    <a href="/my-rides#card-{ride.ID}" class="ride-manage-link inline-block text-sm underline">{m.manageInMyRides()} →</a>
  {:else}
    <ContactOrInterest {ride} {contactPhone} />
  {/if}
  ```

The detail link (`/rides/{ID}`) and the rest of the card are unchanged.

### 2. Own-ride IDs helper — `frontend/src/lib/myRides.ts` (new)

Mirrors the existing `$lib/contacts.ts` (`loadAcceptedContacts`) pattern:

```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '$lib/api';
import type { Ride } from '$lib/types';

// Returns the set of ride IDs published by this phone. Empty set if no phone or
// on error — callers treat "unknown" as "not mine".
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

### 3. Home page wiring — `frontend/src/routes/+page.svelte`

The home page **already** fetches the user's own rides for the pending-interest
badge (`const mine = await api.rides.list({}, phone)`). Reuse that same array to
build the ID set — **no extra request**:

- Add `let myIds = $state<Set<string>>(new Set());`
- Inside the existing `if (phone)` block, set `myIds = new Set(mine.map((r) => r.ID));`
  alongside the existing count logic.
- In the Available tab markup, pass the flag:
  `<RideCard ride={r} contactPhone={contacts.get(r.ID)} isOwn={myIds.has(r.ID)} />`

(The new `loadMyRideIds` helper is **not** used here, to avoid duplicating the
already-in-flight owned-rides request.)

### 4. Search page wiring — `frontend/src/routes/search/+page.svelte`

The search page doesn't fetch own rides today. Add it:

- `import { loadMyRideIds } from '$lib/myRides';`
- Add `let myIds = $state<Set<string>>(new Set());`
- In `run()`, after results load, populate it from the current phone:
  `myIds = await loadMyRideIds(get(userPhone));`
  (also populate once in `onMount` is unnecessary — `run()` is the single results
  path and is called from `onMount` when params exist.)
- Both result columns pass the flag:
  `<RideCard ride={r} contactPhone={contacts.get(r.ID)} isOwn={myIds.has(r.ID)} />`

### 5. i18n — two new keys in all 7 locales

`frontend/src/messages/{fr,en,es,it,de,nl,el}.json`. `manageInMyRides` reuses each
locale's existing "My rides" wording (`btnMyRides`):

| key | fr | en | es | it | de | nl | el |
|---|---|---|---|---|---|---|---|
| `yourRide` | Votre trajet | Your ride | Tu viaje | Il tuo viaggio | Deine Fahrt | Jouw rit | Η διαδρομή σας |
| `manageInMyRides` | Gérer dans Mes trajets | Manage in My rides | Gestionar en Mis viajes | Gestisci in I miei viaggi | In Meine Fahrten verwalten | Beheren in Mijn ritten | Διαχείριση στις διαδρομές μου |

## Data flow (summary)

1. Browse/search page loads public rides (no phone → no driver phone exposed).
2. If a phone is set, the page also obtains the set of the user's own ride IDs
   (home: reuse existing fetch; search: `loadMyRideIds`).
3. Each `RideCard` receives `isOwn = myIds.has(ride.ID)`.
4. Own cards show the "Your ride" badge and a "Manage in My rides →" link instead
   of the contact button.

## Error handling

- No phone / empty profile → `myIds` is empty → nothing marked (normal browse).
- Owned-rides fetch fails → helper returns an empty set; cards render as normal
  public cards (graceful degradation, no error surfaced).

## Testing

- **`RideCard.test.ts`** (exists):
  - `isOwn` → renders the `.tag-your-ride` badge with `m.yourRide()` text, renders
    the manage link with `href="/my-rides#card-<ID>"`, and does **not** render the
    contact button (`.btn-interest` absent).
  - default (`isOwn` false) → renders `ContactOrInterest` (contact button present),
    no `.tag-your-ride`.
- **Search page (`search.test.ts`, exists):** with a mocked `api.rides.list` that
  returns an owned ride (matching ID) for the `X-Phone` call and the same ride in
  results, assert the rendered card shows the "Your ride" badge / manage link.
  (Base-locale `fr` renders by default, so text assertions accept fr/en.)
- **i18n:** both keys present in all 7 locale files; paraglide compiles.

## Files touched

- `frontend/src/lib/components/rides/RideCard.svelte` (prop + badge + manage link)
- `frontend/src/lib/myRides.ts` (new helper)
- `frontend/src/routes/+page.svelte` (build myIds from existing fetch, pass flag)
- `frontend/src/routes/search/+page.svelte` (loadMyRideIds, pass flag)
- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` (two keys)
- `frontend/src/lib/components/rides/RideCard.test.ts` (+ search test) 
