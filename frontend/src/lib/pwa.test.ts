import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { urlBase64ToUint8Array, filterUnseenNotifications, maybeMarkStandalonePrompted } from './pwa';

beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());

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

describe('maybeMarkStandalonePrompted', () => {
	it('returns true once then false (sets the flag)', () => {
		expect(maybeMarkStandalonePrompted()).toBe(true);
		expect(maybeMarkStandalonePrompted()).toBe(false);
		expect(localStorage.getItem('standalone_notif_prompted')).toBe('1');
	});
});
