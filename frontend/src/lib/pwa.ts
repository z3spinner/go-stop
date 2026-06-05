// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { api } from './api';
import { getLocale } from './locale';
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

/**
 * Upsert a push subscription to the backend. Returns success.
 * Exported for testing — the `ON CONFLICT (phone, endpoint) DO UPDATE` upsert
 * makes a repeat call a no-op, which is what lets updateBellState re-post safely.
 */
export async function postSubscription(phone: string, sub: PushSubscription): Promise<boolean> {
	const json = sub.toJSON() as { endpoint?: string; keys?: { p256dh?: string; auth?: string } };
	if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false;
	try {
		await api.subscriptions.upsert({ phone, endpoint: json.endpoint, p256dh: json.keys.p256dh, auth: json.keys.auth });
		return true;
	} catch {
		return false;
	}
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
		return await postSubscription(phone, sub);
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
	if (sub) {
		// The browser's pushManager keeps handing back this subscription even after
		// the push service rotated it (410 Gone) and our server purged the endpoint.
		// Re-post it on every load so the DB recovers; the upsert is a no-op when
		// nothing changed. Without this the bell stays green while pushes silently
		// stop (issue #1).
		if (phone) await postSubscription(phone, sub);
		return pushState.set('subscribed');
	}
	if (phone && (await trySubscribePush(phone))) return pushState.set('subscribed');
	pushState.set('granted');
}

/**
 * Send a test push (the day's localized quote) to the user's devices.
 * Returns how many devices it reached (0 = none registered → suggest reconfigure).
 */
export async function sendTestPush(phone: string): Promise<number> {
	if (!phone) return 0;
	try {
		const res = await api.subscriptions.test(phone, getLocale());
		return res?.sent ?? 0;
	} catch {
		return 0;
	}
}

/**
 * Re-subscribe from scratch: drop the browser's current push subscription and
 * register a fresh one, then refresh the bell. For users who suspect their
 * notifications are broken (e.g. after a push-service rotation).
 */
export async function reconfigurePush(phone: string): Promise<boolean> {
	if (!browser || !('serviceWorker' in navigator)) return false;
	try {
		const reg = await navigator.serviceWorker.ready;
		const existing = await reg.pushManager.getSubscription();
		if (existing) await existing.unsubscribe();
	} catch {
		/* ignore — a fresh subscribe below still recovers most cases */
	}
	const ok = await trySubscribePush(phone);
	await updateBellState(phone);
	return ok;
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
