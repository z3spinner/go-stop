# Repeat a posted ride — design

**Date:** 2026-06-16
**Status:** Approved.

## Goal

When posting a ride, let the driver repeat it a number of times on a cadence
(daily / weekdays / weekly), so a regular trip (e.g. a commute) can be posted in
one go instead of one form submission per day.

## Approach

**Client-side expansion into N independent rides.** No backend or schema change.
The form computes the occurrence dates and posts each as a normal ride via the
existing idempotent `api.rides.post` — the same way the **Return trip** option
already posts a second ride. Each ride remains fully independent: matched,
notified, editable, and auto-deleted at the end of its own day. Re-submitting the
same recurrence is safe (the backend upserts identical rides, no duplicates).

This was chosen over a grouped "recurring series" entity (which would need a new
table, generation logic, and changes to matching/deletion/UI) because the app's
model is already one-off independent rides and this reuses all of it.

## Recurrence semantics

- **Frequencies:**
  - *Daily* — consecutive calendar days (offsets 0, 1, 2, …).
  - *Weekly* — every 7 days, same weekday (offsets 0, 7, 14, …).
  - *Weekdays* — Mon–Fri only; weekends are skipped while counting to N. If the
    entered date is itself a weekend, generation starts at the next weekday (the
    entered date is skipped).
- **Count = total rides**, starting from the date/time entered. Range **1–14**
  (1 = today's behaviour; default when repeating = a small value, e.g. 4).
- Each occurrence is the base shifted by a **whole number of days**;
  **time-of-day is preserved**.
- With **Return trip** also selected, both legs shift by the same day-offset, so
  every occurrence gets its outbound + return (N×2 rides). The out↔return
  relationship the driver entered for the first day is preserved on every day.

## Components

### `frontend/src/lib/recurrence.ts` (new, pure, unit-tested)

```ts
export type Frequency = 'none' | 'daily' | 'weekdays' | 'weekly';

// Returns the whole-day offsets from the base date for each occurrence.
// none      -> [0]
// daily(5)  -> [0,1,2,3,4]
// weekly(3) -> [0,7,14]
// weekdays  -> base stepped day-by-day, counting only Mon–Fri, until `count`
//              collected (a weekend base is skipped, so offset[0] may be > 0).
export function expandOffsets(base: Date, frequency: Frequency, count: number): number[];
```

One clear responsibility (date math), no DOM/network, fully testable.

### `frontend/src/lib/components/rides/RideForm.svelte` (modify)

- Add recurrence state: `frequency: Frequency = 'none'`, `repeatCount = 4`.
- A **Repeat** control after the trip-type toggle: a `<select>` with options
  *Don't repeat* (`none`, default) / *Daily* / *Weekdays* / *Weekly*. When the
  value is not `none`, show a **count** number input (min 1, max 14) and a
  one-line summary: e.g. "Creates 5 rides · Mon 16 Jun → Mon 14 Jul" (first and
  last occurrence dates, localized).
- `submit` becomes: compute `offsets = expandOffsets(new Date(departure_at), frequency, frequency==='none' ? 1 : repeatCount)`; for each offset, post the outbound ride at `departure_at + offset days` and, if `isReturn`, the return at `return_departure_at + offset days`. Posts run **sequentially**; the first error surfaces in `#err` and stops the loop (already-posted rides remain and re-post idempotently on retry).
- Default state and behaviour are unchanged when `frequency === 'none'`.

### i18n (`frontend/src/messages/{7 locales}.json`)

New keys: `repeatLabel`, `repeatNone`, `repeatDaily`, `repeatWeekdays`,
`repeatWeekly`, `repeatCountLabel`, and `repeatSummary` (with `{count}`,
`{first}`, `{last}` placeholders).

## Error handling

- Sequential posts; on the first failure the loop stops and the error message is
  shown in `#err`. Rides already created stay (idempotent re-post on retry).
- The count input is clamped to 1–14; non-repeat path posts exactly one ride
  (and one return), identical to current behaviour.

## Accepted trade-offs

1. Recurring rides show as separate cards in My Rides (deleted/edited
   individually) — no grouping.
2. A searcher with a matching route alert receives one notification per
   occurrence (each is a genuinely different date/travel option).

## Testing

- **`recurrence.test.ts`** — unit tests for `expandOffsets`: daily/weekly offset
  sequences; count = total (count 1 → `[0]`); weekdays skips weekends and a
  weekend base; count clamping is the form's job, not the helper's.
- **`RideForm` component test** — with `api.rides.post` mocked: selecting *Daily*
  + count 3 posts 3 rides with dates +0/+1/+2 days at the same time; with Return
  on, posts 3 outbound + 3 return; `none` posts exactly one (regression).

## Out of scope (YAGNI)

- Grouped/recurring-series entity or "edit/delete all" actions.
- Backend batch endpoint (client loop reuses the idempotent post).
- Arbitrary RRULE-style rules, end-date pickers, or "every N weeks".
