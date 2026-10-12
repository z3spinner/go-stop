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
