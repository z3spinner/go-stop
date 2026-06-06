// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { api } from '$lib/api';
import type { Ride } from '$lib/types';

/**
 * Returns the set of ride IDs published by this phone. Empty set when no phone is
 * set or the request fails — callers treat "unknown" as "not mine".
 */
export async function loadMyRideIds(phone: string): Promise<Set<string>> {
	if (!phone) return new Set();
	try {
		const mine = (await api.rides.list({}, phone)) as Ride[];
		return new Set(mine.map((r) => r.ID));
	} catch {
		return new Set();
	}
}
