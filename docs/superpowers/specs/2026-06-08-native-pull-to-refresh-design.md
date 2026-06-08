# Native-feel pull-to-refresh — design

**Date:** 2026-06-08
**Status:** Approved

## Problem

The custom standalone-iOS pull-to-refresh works but doesn't feel native: the page
content stays put (only a floating badge appears) and the spinner is a
Material-style ring. Native iOS drags the content down, revealing the indicator
in the opening gap, with the segmented "spoke" activity indicator.

## Design

`PullToRefresh.svelte` becomes a thin **wrapper** around the page content.

### Content tracks the finger
- The component renders its children inside a `.ptr-track` div with
  `transform: translateY(offset)`.
- `offset = pull` while dragging (damped ×0.5, capped at 110px); `= 56px` (a rest
  height) while refreshing; eases to 0 on release/settle.
- During an active drag the transform has **no transition** (1:1 with the finger);
  on release/settle a spring-ish `transition: transform 0.3s cubic-bezier(0.2,0.8,0.2,1)`.
- Gesture logic is unchanged from the current version: arm only at
  `scrollY === 0`, lock to vertical (let horizontal swipes through),
  `preventDefault` on the non-passive `touchmove` to suppress rubber-band,
  standalone-only (browser tabs keep native PTR), `THRESHOLD=70`, `DAMP=0.5`.
- `children` is an **optional** snippet, so the existing gesture unit tests
  (which render with no children) keep working.

### iOS "spoke" activity indicator
- 12 tapered, rounded spokes arranged radially (30° apart) with a fading opacity
  gradient, positioned just above the content (`top: -indicatorHeight`) so it
  descends into the revealed gap as the content pulls down (like UIRefreshControl,
  which scrolls with the content).
- **Pulling:** spokes reveal in proportion to `progress = pull/THRESHOLD` (fill
  clockwise), and the whole indicator fades in.
- **Refreshing:** the indicator spins continuously (CSS rotation) with the static
  opacity gradient — the classic chasing-fade look.
- Brand-green tint.

### Layout change
`+layout.svelte` wraps **header + `#app` (the `{#key}`-ed page) + footer** inside
`<PullToRefresh onrefresh={refresh}> … </PullToRefresh>`. Modals/toasts stay
**outside** the wrapper so overlays aren't translated.

Caveat: a `transform` on the wrapper makes it the containing block for any
`position: fixed` descendants — but the content area has none (status-bar strip,
toasts, modals all live outside the wrapper), and `offset` is 0 except mid-gesture.

## Verification
- Existing gesture unit tests stay green (trigger logic unchanged).
- `svelte-check` + full frontend test suite pass.
- Visual: run the dev stack, force standalone, drive synthetic touch in the
  browser, and screenshot the pulled + refreshing states to confirm the native
  feel (content drags down, spoke indicator). Reverted before commit.

## Out of scope
- Browser-tab behavior (unchanged — native gesture).
- Per-page refresh customization (still the layout-wide `invalidateAll` + remount).
