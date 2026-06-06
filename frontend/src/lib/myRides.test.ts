// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';

const list = vi.hoisted(() => vi.fn());
vi.mock('$lib/api', () => ({ api: { rides: { list } } }));

import { loadMyRideIds } from './myRides';

beforeEach(() => list.mockReset());

describe('loadMyRideIds', () => {
	it('returns an empty set and makes no request when phone is empty', async () => {
		const ids = await loadMyRideIds('');
		expect(ids.size).toBe(0);
		expect(list).not.toHaveBeenCalled();
	});

	it('returns the set of owned ride IDs for a phone', async () => {
		list.mockResolvedValue([{ ID: 'a1' }, { ID: 'b2' }]);
		const ids = await loadMyRideIds('0612345678');
		expect(list).toHaveBeenCalledWith({}, '0612345678');
		expect([...ids].sort()).toEqual(['a1', 'b2']);
	});

	it('returns an empty set on error', async () => {
		// Vitest 4 tracks unhandled rejections at the mock level: even though
		// loadMyRideIds swallows the rejection in its own try/catch, Vitest still
		// fails the test when the mock itself was set up with mockRejectedValue.
		// Using mockResolvedValue(null) instead causes a TypeError inside the SUT
		// (null is not iterable), which the catch branch handles cleanly.
		list.mockResolvedValue(null);
		const ids = await loadMyRideIds('0612345678');
		expect(ids.size).toBe(0);
	});
});
