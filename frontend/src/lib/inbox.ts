// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// In-app notification inbox: every push a user receives is also kept on the
// server for a month (GET /api/inbox), so a missed or swiped-away notification
// can still be read here. The bell shows the unread count; /notifications lists
// them. Text is rendered from the item's `kind` in the reader's language.

import { writable } from 'svelte/store';
import { api } from './api';
import { m } from './paraglide/messages';
import type { InboxItem } from './types';

/** Server default for NOTIFICATION_RETENTION_DAYS, used until /config loads. */
export const DEFAULT_RETENTION_DAYS = 30;

export const inboxItems = writable<InboxItem[]>([]);
export const inboxUnread = writable(0);

const THROTTLE_MS = 30_000;
let lastLoadMs = 0;

/**
 * Reload the inbox for phone. Calls within 30s of the previous load are skipped
 * unless `force` (used when a push just arrived or the inbox page opens).
 * Network errors keep the last known state.
 */
export async function refreshInbox(phone: string, { force = false } = {}): Promise<void> {
	if (!phone) {
		inboxItems.set([]);
		inboxUnread.set(0);
		return;
	}
	const now = Date.now();
	if (!force && now - lastLoadMs < THROTTLE_MS) return;
	lastLoadMs = now;
	try {
		const res = await api.inbox.list(phone);
		inboxItems.set(res.items ?? []);
		inboxUnread.set(res.unread ?? 0);
	} catch {
		/* offline / server error: keep what we have */
	}
}

/** Mark everything read (clears the bell badge). Best-effort. */
export async function markAllInboxRead(phone: string): Promise<void> {
	if (!phone) return;
	try {
		await api.inbox.markRead(phone);
		inboxUnread.set(0);
	} catch {
		/* the badge just stays until the next successful refresh */
	}
}

type TitleFn = (name: string) => string;

const TITLES: Record<string, TitleFn> = {
	ride_match: (name) => m.inboxKindRideMatch({ name }),
	request_match: (name) => m.inboxKindRequestMatch({ name }),
	interest_received: (name) => m.inboxKindInterestReceived({ name }),
	interest_cancelled: (name) => m.inboxKindInterestCancelled({ name }),
	interest_accepted: (name) => m.inboxKindInterestAccepted({ name }),
	driver_shared: (name) => m.inboxKindDriverShared({ name }),
	contact_offered: (name) => m.inboxKindContactOffered({ name })
};

const ICONS: Record<string, string> = {
	ride_match: '🚗',
	request_match: '🙋',
	interest_received: '📩',
	interest_cancelled: '↩️',
	interest_accepted: '✅',
	driver_shared: '🚗',
	contact_offered: '🤝',
	feedback_reminder: '💬'
};

/** Localized headline for an item; falls back to the server text for unknown kinds. */
export function inboxTitle(item: InboxItem): string {
	if (item.kind === 'feedback_reminder') return m.inboxKindFeedbackReminder();
	const fn = TITLES[item.kind];
	if (fn && item.contact_name) return fn(item.contact_name);
	return item.title || m.inboxTitle();
}

/** "Origin → Destination", or the server body when the route is unknown. */
export function inboxRoute(item: InboxItem): string {
	if (item.origin && item.destination) return `${item.origin} → ${item.destination}`;
	return item.body;
}

export function inboxIcon(item: InboxItem): string {
	return ICONS[item.kind] ?? '🔔';
}

/** True when the trip this item is about has already departed. */
export function isPastTrip(item: InboxItem, now: number = Date.now()): boolean {
	if (!item.departure_at) return false;
	const t = Date.parse(item.departure_at);
	return !Number.isNaN(t) && t < now;
}

/** Bell badge text: hidden at 0, capped at "9+". */
export function badgeText(unread: number): string {
	if (unread <= 0) return '';
	return unread > 9 ? '9+' : String(unread);
}
