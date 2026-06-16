# Modernise form controls (combobox / select / number stepper) — design

**Date:** 2026-06-16
**Status:** Approved.

## Goal

Replace the app's ad-hoc native form controls — `<input list=datalist>`
comboboxes, `<select>`s, and the `<input type=number>` stepper — with mature,
consistently-styled components from the library the project already ships
(**shadcn-svelte 1.3 on bits-ui 2.18**). The native controls behave
inconsistently across browsers; the replacements give proper keyboard nav,
filtering, and brand-consistent styling.

No new dependency: bits-ui and the shadcn `ui/select` wrapper are already
installed; only `ui/combobox` (bits-ui Combobox) needs to be added.

## Scope

**Convert:**
- All origin/destination `<input list=datalist>` autocompletes (8 instances:
  `RideForm`, `AlertForm`, `search/+page`, `rides/[id]/edit`).
- All flexibility `<select>`s (Exact / ±30 / ±60 — 4 instances: `RideForm` ×2,
  `AlertForm`, `rides/[id]/edit`).
- The repeat **frequency** `<select>` (`RideForm`).
- The repeat **count** `<input type=number>` (`RideForm`).

**Keep native (out of scope):** all date / time / `datetime-local` inputs, the
trip-type toggle, and the alert-mode controls. (User flagged
selects/datalists/number inputs; date pickers are a separate, larger swap.)

## Reusable components (the DRY core)

### `PlaceCombobox.svelte` — free-text autocomplete (bits-ui Combobox)

The most important and subtle piece. Origin/destination are **free-form**: the
driver may type any place; the `destinations` list is only suggestions. So this
is an *autocomplete that allows arbitrary values*, not a constrained select.

**Contract (props):**
- `value: string` (bindable) — the committed text. **This is the typed input
  text**, not a selected item id.
- `items: string[]` — suggestion source (the `destinations` array).
- `name`, `id`, `placeholder`, `required`, `disabled` — passthrough for the
  underlying input (so existing `input[name=origin]` selectors and form
  semantics survive).

**Behaviour:**
- Typing filters `items` (case/accent-insensitive substring) into the dropdown.
- Picking a suggestion sets `value` to that suggestion.
- Typing a value **not** in the list and blurring/submitting keeps the typed
  text as `value` (free text is valid).
- `value` is always the current input text — bound via bits-ui Combobox's
  `inputValue`. We do not depend on bits-ui's item-selection `value`.
- Renders a real `<input>` carrying `name`/`required` so HTML form behaviour and
  the existing tests' `input[name=...]` selectors keep working.

Built on `bits-ui` `Combobox` (Root/Input/Trigger/Portal/Content/Viewport/Item)
+ the shadcn select-content/item styling wrappers. Implementer consults the
bits-ui Combobox docs (context7 `huntabyte/bits-ui`) for exact wiring; the
component tests below pin the required behaviour.

### `FlexibilitySelect.svelte` — shadcn Select

Wraps `$lib/components/ui/select`. Props: `value: number` (bindable; 0/30/60).
Renders the three options via `m.flexExact()/m.flex30()/m.flex60()`. Replaces
the 4 flexibility `<select>`s.

### `NumberStepper.svelte` — − / value / +

Built from shadcn `ui/button` + `ui/input`. Props: `value: number` (bindable),
`min`, `max`. Minus/plus buttons step by 1 and are `disabled` at the bounds; the
input is read-only-ish (typing clamps to `[min,max]` on change). Used for the
repeat count (min 1, max 14).

### Frequency dropdown

`RideForm`'s frequency uses `ui/select` directly (single use) with the four
`m.repeat*` options — no dedicated wrapper.

## Theming

The shadcn components reference the `--primary` / `--input` / `--ring` tokens
already defined (green brand) in `app.css`, the same tokens used to restyle the
Tabs. The trigger/content/input get class overrides to match the app's existing
input look (border, `rounded`, green focus ring). Verified visually in the
devstack.

## Per-form changes (mechanical, once the components exist)

- `RideForm`: origin/dest → `PlaceCombobox`; flexibility ×2 → `FlexibilitySelect`;
  frequency → `ui/select`; count → `NumberStepper`.
- `AlertForm`: origin/dest → `PlaceCombobox`; flexibility → `FlexibilitySelect`.
- `search/+page`: origin/dest → `PlaceCombobox`.
- `rides/[id]/edit`: origin/dest → `PlaceCombobox`; flexibility → `FlexibilitySelect`.

The `<datalist>` elements and their `list=` attributes are removed.

## Testing

- **`PlaceCombobox` test** — binding round-trips typed text to `value`; typing
  filters suggestions; clicking a suggestion sets `value`; a free-text value not
  in `items` is preserved; the rendered input carries `name`/`required`.
- **`NumberStepper` test** — `+`/`−` step within `[min,max]`; buttons disabled at
  bounds; out-of-range typed input clamps.
- **`FlexibilitySelect` test** — selecting an option updates the bound value.
- **Existing form/route tests** stay green. Where a test currently drives a
  native control directly (e.g. sets a `<select>`/datalist input value), update
  the selector/interaction to the new component while keeping the *assertions*
  (posted/PUT payloads) unchanged. The form tests assert API payloads, so the
  observable contract is preserved.

## Risks / mitigations

- **Combobox free-text** is the main risk — bits-ui Combobox is selection-first.
  Mitigation: drive the form value from `inputValue`, treat items purely as
  suggestions, and pin behaviour with the component test. If bits-ui cannot
  cleanly support free-text commit, fall back to a thin custom autocomplete
  (input + filtered listbox) — but try bits-ui first per the approved direction.
- **Portal/overlay in forms** (Content renders in a portal): verify it opens,
  positions, and closes correctly inside the existing `PullToRefresh` layout in
  the devstack.

## Out of scope (YAGNI)

- Date/time/`datetime-local` → bits-ui DatePicker/DateField (separate effort).
- Multi-select, async-loaded options, virtualised lists.
