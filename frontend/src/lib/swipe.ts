// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// Swipe-to-delete decision, kept pure so it can be unit-tested. A row is
// deleted when it is dragged far enough, or flicked quickly, in either
// direction — like a phone's notification shade.

/** Fraction of the row's width that deletes on release. */
export const SWIPE_DISTANCE = 0.4;
/** A flick needs at least this many px … */
export const FLICK_MIN_PX = 40;
/** … at at least this speed (px per ms). */
export const FLICK_SPEED = 0.5;
/** Movement before we decide between a horizontal swipe and a vertical scroll. */
export const DIRECTION_SLOP = 8;

/**
 * What a released drag should do: -1 (delete, leaving left), 1 (delete,
 * leaving right) or 0 (snap back).
 */
export function swipeOutcome(dx: number, width: number, ms: number): -1 | 0 | 1 {
	const dist = Math.abs(dx);
	if (dist === 0) return 0;
	const dir = dx < 0 ? -1 : 1;
	if (width > 0 && dist >= width * SWIPE_DISTANCE) return dir;
	if (dist >= FLICK_MIN_PX && ms > 0 && dist / ms >= FLICK_SPEED) return dir;
	return 0;
}

/**
 * Once movement passes the slop, whether the gesture is a horizontal swipe
 * ('h'), a vertical scroll ('v'), or still undecided (null).
 */
export function swipeDirection(dx: number, dy: number): 'h' | 'v' | null {
	if (Math.abs(dx) < DIRECTION_SLOP && Math.abs(dy) < DIRECTION_SLOP) return null;
	return Math.abs(dx) > Math.abs(dy) ? 'h' : 'v';
}
