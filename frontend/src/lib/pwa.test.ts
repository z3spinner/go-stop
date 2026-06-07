// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	urlBase64ToUint8Array,
	filterUnseenNotifications,
	postSubscription,
	updateBellState,
	pushState
} from './pwa';
import { api } from './api';

// updateBellState bails out under SSR (`browser === false`); force it true so the
// re-post path runs. The pure-function tests below don't read `browser`.
vi.mock('$app/environment', () => ({ browser: true }));

beforeEach(() => localStorage.clear());
afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

/** A minimal PushSubscription whose toJSON() yields the given shape. */
const fakeSub = (json: unknown) => ({ toJSON: () => json }) as unknown as PushSubscription;

describe('urlBase64ToUint8Array', () => {
	it('decodes a base64url VAPID key to bytes', () => {
		const out = urlBase64ToUint8Array('AQID'); // base64 for [1,2,3]
		expect(Array.from(out)).toEqual([1, 2, 3]);
	});
});

describe('filterUnseenNotifications', () => {
	it('returns only ride_ids not already in poll_seen and records them', () => {
		localStorage.setItem('poll_seen', JSON.stringify(['r1']));
		const incoming = [{ ride_id: 'r1' }, { ride_id: 'r2' }] as any;
		const fresh = filterUnseenNotifications(incoming);
		expect(fresh.map((n: any) => n.ride_id)).toEqual(['r2']);
		expect(JSON.parse(localStorage.getItem('poll_seen')!)).toContain('r2');
	});
});

describe('postSubscription', () => {
	it('upserts a complete subscription and returns true', async () => {
		const upsert = vi.spyOn(api.subscriptions, 'upsert').mockResolvedValue(null);
		const ok = await postSubscription('0600000001', fakeSub({ endpoint: 'https://push/x', keys: { p256dh: 'p', auth: 'a' } }));
		expect(ok).toBe(true);
		expect(upsert).toHaveBeenCalledWith({ phone: '0600000001', endpoint: 'https://push/x', p256dh: 'p', auth: 'a' });
	});

	it('does not post and returns false when keys are missing', async () => {
		const upsert = vi.spyOn(api.subscriptions, 'upsert').mockResolvedValue(null);
		const ok = await postSubscription('0600000001', fakeSub({ endpoint: 'https://push/x', keys: {} }));
		expect(ok).toBe(false);
		expect(upsert).not.toHaveBeenCalled();
	});

	it('swallows upsert errors and returns false', async () => {
		vi.spyOn(api.subscriptions, 'upsert').mockRejectedValue(new Error('network'));
		const ok = await postSubscription('0600000001', fakeSub({ endpoint: 'https://push/x', keys: { p256dh: 'p', auth: 'a' } }));
		expect(ok).toBe(false);
	});
});

describe('updateBellState (issue #1: re-post on every load)', () => {
	// Stub just enough of the browser push surface for updateBellState to reach
	// the getSubscription() branch.
	function stubPushEnv(getSubscription: () => unknown) {
		vi.stubGlobal('Notification', { permission: 'granted' });
		vi.stubGlobal('matchMedia', () => ({ matches: false }));
		vi.stubGlobal('navigator', {
			userAgent: 'vitest',
			serviceWorker: { ready: Promise.resolve({ pushManager: { getSubscription } }) }
		});
	}

	it('re-posts the existing subscription so the DB recovers after a 410 rotation', async () => {
		const upsert = vi.spyOn(api.subscriptions, 'upsert').mockResolvedValue(null);
		const sub = fakeSub({ endpoint: 'https://push/old', keys: { p256dh: 'p', auth: 'a' } });
		stubPushEnv(async () => sub);

		await updateBellState('0600000001');

		expect(upsert).toHaveBeenCalledWith({ phone: '0600000001', endpoint: 'https://push/old', p256dh: 'p', auth: 'a' });
		expect(get(pushState)).toBe('subscribed');
	});

	it('reports subscribed without posting when no phone is known', async () => {
		const upsert = vi.spyOn(api.subscriptions, 'upsert').mockResolvedValue(null);
		stubPushEnv(async () => fakeSub({ endpoint: 'https://push/old', keys: { p256dh: 'p', auth: 'a' } }));

		await updateBellState();

		expect(upsert).not.toHaveBeenCalled();
		expect(get(pushState)).toBe('subscribed');
	});
});
