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
		// Return a non-iterable value so the internal map() call throws, exercising
		// the catch branch without creating a Vitest-tracked unhandled rejection.
		list.mockResolvedValue(null);
		const ids = await loadMyRideIds('0612345678');
		expect(ids.size).toBe(0);
	});
});
