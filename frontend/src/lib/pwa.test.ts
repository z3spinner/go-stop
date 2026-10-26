// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	isInAppBrowser,
	urlBase64ToUint8Array,
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

describe('isInAppBrowser', () => {
	const safari = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1';
	const webkit = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148';
	it('spots the browsers built into Facebook, Messenger, Instagram and others', () => {
		expect(isInAppBrowser(`${webkit} [FBAN/FBIOS;FBAV/530.0.0.40.97;FBBV/1;FBDV/iPhone12,8;FBMD/iPhone;FBSN/iOS]`)).toBe(true);
		expect(isInAppBrowser(`${webkit} [FBAN/MessengerForiOS;FBAV/500.0.0.32.106]`)).toBe(true);
		expect(isInAppBrowser(`${webkit} Instagram 400.0.0.25.86 (iPhone12,8; iOS 18_6; fr_FR)`)).toBe(true);
		expect(isInAppBrowser(`${webkit} WhatsApp/25.20.80`)).toBe(true);
		expect(isInAppBrowser(`${webkit} LinkedInApp/9.31`)).toBe(true);
		expect(isInAppBrowser(`${webkit} Snapchat/13.50.0.38`)).toBe(true);
		expect(isInAppBrowser(`${webkit} musical_ly_41.0.0 BytedanceWebview/d8a21c6`)).toBe(true);
	});
	it('leaves Safari and other real browsers alone', () => {
		expect(isInAppBrowser(safari)).toBe(false);
		expect(isInAppBrowser(`${webkit} CriOS/141.0.7390.96 Mobile/15E148 Safari/604.1`)).toBe(false);
		expect(isInAppBrowser(`${webkit} FxiOS/144.0 Mobile/15E148 Safari/605.1.15`)).toBe(false);
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
