// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { swipeDirection, swipeOutcome } from './swipe';

describe('swipeOutcome', () => {
	it('deletes past 40% of the width, in the swipe direction', () => {
		expect(swipeOutcome(-160, 400, 1000)).toBe(-1);
		expect(swipeOutcome(170, 400, 1000)).toBe(1);
	});
	it('snaps back for a short, slow drag', () => {
		expect(swipeOutcome(-100, 400, 1000)).toBe(0);
		expect(swipeOutcome(0, 400, 50)).toBe(0);
	});
	it('deletes on a quick flick even when short', () => {
		expect(swipeOutcome(-60, 400, 80)).toBe(-1);
	});
	it('ignores tiny fast jitters', () => {
		expect(swipeOutcome(20, 400, 10)).toBe(0);
	});
});

describe('swipeDirection', () => {
	it('waits until movement passes the slop', () => {
		expect(swipeDirection(3, 4)).toBeNull();
	});
	it('locks horizontal or vertical', () => {
		expect(swipeDirection(-20, 5)).toBe('h');
		expect(swipeDirection(4, 30)).toBe('v');
	});
});
