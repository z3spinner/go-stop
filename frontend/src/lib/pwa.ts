import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { api } from './api';
import type { NotificationItem } from './types';

// True in a real browser and in the jsdom test environment, false during SSR /
// prerender. We guard localStorage-only functions on the actual presence of
// `localStorage` rather than SvelteKit's `browser` flag, because that flag is
// compiled to `false` under Vitest and would disable pure functions in tests.
// See locale.ts for the same pattern.
function hasStorage(): boolean {
	return typeof localStorage !== 'undefined';
}

export type PushState = 'unsupported' | 'ios-browser' | 'default' | 'granted' | 'denied' | 'subscribed';

export const pushState = writable<PushState>('default');
/** Notifications surfaced by polling; the layout renders a PollToast per item. */
export const pollToasts = writable<NotificationItem[]>([]);

export function isIOSBrowser(): boolean {
	if (!browser) return false;
	const ua = navigator.userAgent;
	const isIOS = /iPad|iPhone|iPod/.test(ua) && !(window as unknown as { MSStream?: unknown }).MSStream;
	const standalone =
		(navigator as unknown as { standalone?: boolean }).standalone === true ||
		window.matchMedia('(display-mode: standalone)').matches;
	return isIOS && !standalone;
}

export function isStandalone(): boolean {
	if (!browser) return false;
	return (
		(navigator as unknown as { standalone?: boolean }).standalone === true ||
		window.matchMedia('(display-mode: standalone)').matches
	);
}

export function urlBase64ToUint8Array(base64String: string): Uint8Array {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const raw = atob(base64);
	const out = new Uint8Array(raw.length);
	for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
	return out;
}

/** Subscribe to Web Push and register with the backend. Returns success. */
export async function trySubscribePush(phone: string): Promise<boolean> {
	if (!browser || !('serviceWorker' in navigator) || !('PushManager' in window)) return false;
	try {
		const reg = await navigator.serviceWorker.ready;
		const { publicKey } = await api.vapid.getPublicKey();
		if (!publicKey) return false;
		const sub = await reg.pushManager.subscribe({
			userVisibleOnly: true,
			applicationServerKey: urlBase64ToUint8Array(publicKey) as unknown as BufferSource
		});
		const json = sub.toJSON() as { endpoint?: string; keys?: { p256dh?: string; auth?: string } };
		if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false;
		await api.subscriptions.upsert({ phone, endpoint: json.endpoint, p256dh: json.keys.p256dh, auth: json.keys.auth });
		return true;
	} catch {
		return false;
	}
}

/** Recompute the bell/push state; silently re-subscribe if a subscription expired. */
export async function updateBellState(phone?: string): Promise<void> {
	if (!browser) return;
	if (isIOSBrowser()) return pushState.set('ios-browser');
	if (!('Notification' in window) || !('serviceWorker' in navigator)) return pushState.set('unsupported');
	const perm = Notification.permission;
	if (perm === 'denied') return pushState.set('denied');
	if (perm !== 'granted') return pushState.set('default');
	const reg = await navigator.serviceWorker.ready;
	const sub = await reg.pushManager.getSubscription();
	if (sub) return pushState.set('subscribed');
	if (phone && (await trySubscribePush(phone))) return pushState.set('subscribed');
	pushState.set('granted');
}

const SEEN_KEY = 'poll_seen';

/** Pure: given incoming notifications, return those whose ride_id is unseen and record them (cap 100). */
export function filterUnseenNotifications(items: NotificationItem[]): NotificationItem[] {
	if (!hasStorage()) return items;
	let seen: string[] = [];
	try {
		seen = JSON.parse(localStorage.getItem(SEEN_KEY) ?? '[]');
	} catch {
		seen = [];
	}
	const fresh = items.filter((n) => !seen.includes(n.ride_id));
	if (fresh.length) {
		const next = [...seen, ...fresh.map((n) => n.ride_id)].slice(-100);
		localStorage.setItem(SEEN_KEY, JSON.stringify(next));
	}
	return fresh;
}

let lastPollMs = 0;

/** Poll the backend for ride-match notifications and push fresh ones to pollToasts (throttled 60s). */
export async function pollForNotifications(phone: string): Promise<void> {
	if (!browser || !phone) return;
	const now = Date.now();
	if (now - lastPollMs < 60_000) return;
	lastPollMs = now;
	try {
		const items = await api.notifications.list(phone);
		const fresh = filterUnseenNotifications(items);
		if (fresh.length) pollToasts.update((t) => [...t, ...fresh]);
	} catch {
		/* network errors are non-fatal for polling */
	}
}

/** Returns true the first time only (sets standalone_notif_prompted). */
export function maybeMarkStandalonePrompted(): boolean {
	if (!hasStorage()) return false;
	if (localStorage.getItem('standalone_notif_prompted')) return false;
	localStorage.setItem('standalone_notif_prompted', '1');
	return true;
}
