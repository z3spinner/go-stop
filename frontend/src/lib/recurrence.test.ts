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
