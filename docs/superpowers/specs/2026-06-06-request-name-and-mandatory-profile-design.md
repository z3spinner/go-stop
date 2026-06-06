# Show requester name + make profile (name & phone) mandatory before requesting contact

**Date:** 2026-06-06
**Status:** Approved (design)

## Problem

On the driver's "My rides" view (`MyRideCard.svelte`), when someone requests their
contact ("expresses interest"):

1. The **incoming request** shows the searcher's name only as a bare, unlabeled
   string — and it is usually empty, because providing a name is optional today.
2. After the driver **accepts**, the name is dropped entirely; only the phone is
   shown ("Contact accepted: 0612…").

The root cause of the empty names is that a name is **not required** anywhere in
the request-contact flow:

- Frontend `ContactOrInterest.svelte` sends the name only `|| undefined`, and the
  fallback `window.prompt` asks for a *phone* only — never a name.
- Backend `expressInterestRequest` binds `Name` with no `required` constraint; the
  usecase and DB store an empty string (`searcher_name … DEFAULT ''`).

A blank name also corrupts the driver's push notification, which renders as
" cherche un trajet …" with a leading space where the name should be.

## Goals

- Driver always sees a clear, labeled requester name — for incoming requests and
  after accepting.
- A name **and** a phone become mandatory (held in the user's profile) before a
  user can request a contact. Close the hole at the source.
- Enforce the name requirement on the server too (defense in depth + fixes the
  notification text).

## Non-goals

- No change to the "matching alerts / seekers" path (`SeekerRow` already shows a
  name). Scope is the *interest / request-contact* flow only.
- No accounts/auth changes — profile remains client-side localStorage
  (`userName`, `userPhone` stores), consistent with the app's existing model.
- No backfill of existing empty-name interests (handled by a display placeholder).

## Design

### 1. Display the name (driver's view) — `MyRideCard.svelte`

Use labeled phrases, per agreed UX:

- **Pending** request row: `{name} wants a ride` followed by the **Accept** button.
- **Accepted** row: `{name} — {phone}` (phone remains a `tel:` link).

`searcher_name` is already present on `InterestListItem` and is preserved
client-side after accepting (the accept handler spreads `...x`), so **no backend
change is needed for display** — only the row templates change.

**Legacy / empty name:** rows created before this change may have an empty
`searcher_name`. Render a neutral placeholder ("Someone") in that case so a row is
never blank, e.g. `it.searcher_name?.trim() || m.anonymousSearcher()`.

New i18n message keys (added to all 7 locales: fr [base], en, es, it, de, nl, el):

- `interestPendingName` — `"{name} wants a ride"` (and locale equivalents)
- `anonymousSearcher` — `"Someone"`
- Accepted row reuses the name + existing phone link; if a separate key reads
  better per locale, add `contactAcceptedName` = `"{name}"` — otherwise compose
  inline. (Implementation may keep `contactRevealed` for the empty-name fallback.)

### 2. Gate requesting a contact behind a complete profile — Profile modal

A new reusable modal prompts for name + phone when a user with an incomplete
profile taps "Request contact", then continues the original action.

**New store** — `src/lib/profileModal.ts`:

```ts
import { writable } from 'svelte/store';
// Holds the action to run once the profile is completed, or null when closed.
export const profileModalState = writable<(() => void) | null>(null);
export function openProfileModal(onComplete: () => void) {
  profileModalState.set(onComplete);
}
```

**New component** — `src/lib/components/profile/ProfileModal.svelte`:

- Built on the same `$lib/components/ui/dialog` primitives as `NotifModal`.
- Fields: name + phone, bound to a local copy (seeded from `userName`/`userPhone`).
- A "Save & continue" button, disabled until both fields are non-empty
  (`name.trim()` and `phone.trim()`), mirroring `NotifModal`'s `profileComplete`.
- On save: `userName.set(name.trim())`, `userPhone.set(normalizePhone(phone))`,
  then run the stored `onComplete` callback and close.
- Mounted once in `src/routes/+layout.svelte` alongside `<NotifModal />`.

New i18n keys: `profileRequiredTitle`, `profileRequiredBody`, `btnSaveContinue`
(all 7 locales). Reuse existing `labelName` / `labelPhone`.

**Wire into `ContactOrInterest.svelte`** — replace the phone-only `window.prompt`
hack in `express()`:

```ts
async function express() {
  if (busy) return;
  const name = get(userName).trim();
  const phone = get(userPhone).trim();
  if (!name || !phone) {
    openProfileModal(() => express()); // re-run once the profile is complete
    return;
  }
  busy = true;
  // … existing call, now always sending the (required) name:
  const res = await api.interests.express(ride.ID, normalizePhone(phone), name);
  // … unchanged
}
```

`get(userName)` / `get(userPhone)` are read fresh on re-entry, so the recursive
call after the modal saves will pass the gate.

### 3. Enforce on the server — `interest_handler.go` + usecase

- Add `binding:"required"` to `Name` in `expressInterestRequest`, **and** reject a
  whitespace-only name after `strings.TrimSpace`, returning `400` with a clear
  error (binding alone won't catch `"   "`).
- Optionally also guard in `ExpressInterest.Execute` (return an error on empty
  name) so the invariant holds regardless of caller. Keep the existing
  driver/not-found error mapping.

This also fixes the notification body — `searcherName` is now guaranteed non-empty.

## Data flow (request-contact, after change)

1. User taps "Request contact" in `ContactOrInterest`.
2. If profile incomplete → `ProfileModal` opens → user enters name+phone → saved to
   `userName`/`userPhone` stores → `onComplete` re-runs `express()`.
3. `express()` POSTs `{ phone, name }` (both non-empty) to `/rides/{id}/interest`.
4. Backend validates non-empty name (else `400`), stores the interest, notifies the
   driver with a correctly-formed body.
5. Driver's `MyRideCard` shows `"{name} wants a ride" [Accept]`; after accepting,
   `"{name} — {phone}"`.

## Error handling

- Empty/whitespace name client-side: cannot reach the API — gated by the modal.
- Empty name reaching the API anyway: `400`, surfaced inline via existing
  `stateMsg` error handling in `ContactOrInterest`.
- Modal dismissed without completing: no request is sent; button returns to idle.

## Testing

- **Frontend (vitest):**
  - `ContactOrInterest`: incomplete profile opens the modal and does *not* call
    `api.interests.express`; completing the modal then sends with a non-empty name.
    (Extend existing `ContactOrInterest.test.ts`.)
  - `MyRideCard`: pending row renders `"{name} wants a ride"`; accepted row renders
    `"{name} — {phone}"`; empty name renders the placeholder.
  - `ProfileModal`: "Save & continue" disabled until both fields filled; saving
    persists to stores and fires `onComplete`.
- **Backend (Go):** `Express` returns `400` for empty / whitespace-only name;
  succeeds (`201`) with a valid name.
- **i18n:** all new keys present in every locale file (the repo's existing locale
  completeness check / build must pass).

## Files touched

- `frontend/src/lib/components/rides/MyRideCard.svelte` (display)
- `frontend/src/lib/components/rides/ContactOrInterest.svelte` (gate)
- `frontend/src/lib/profileModal.ts` (new store)
- `frontend/src/lib/components/profile/ProfileModal.svelte` (new component)
- `frontend/src/routes/+layout.svelte` (mount modal)
- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` (new keys)
- `internal/boundaries/handler/interest_handler.go` (required name)
- `internal/usecase/express_interest.go` (optional guard)
- Tests alongside the above.
