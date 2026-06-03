import { browser } from '$app/environment';
import { api } from './api';
import type { MyInterest } from './types';

/**
 * For the given phone, fetch the searcher's own interests, sync accepted/driver_shared
 * ones into localStorage (interest_<rideId>), and return a map rideId -> contact phone.
 */
export async function loadAcceptedContacts(phone: string): Promise<Map<string, string>> {
	const map = new Map<string, string>();
	if (!phone) return map;
	let mine: MyInterest[] = [];
	try {
		mine = await api.interests.listMine(phone);
	} catch {
		return map;
	}
	for (const it of mine) {
		if (browser) localStorage.setItem(`interest_${it.ride_id}`, it.id);
		if (it.status === 'accepted' || it.status === 'driver_shared') {
			try {
				const c = await api.interests.getContact(it.id, phone);
				map.set(it.ride_id, c.phone);
			} catch {
				/* ignore */
			}
		}
	}
	return map;
}
