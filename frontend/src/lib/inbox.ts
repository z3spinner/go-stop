// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// In-app notification inbox: every push a user receives is also kept on the
// server (GET /api/inbox), so a missed or swiped-away notification can still be
// read here. The bell shows the unread count; /notifications lists them. A
// notification is read only once it is tapped — in the list or as a push — not
// because it was on screen. Text is rendered from the item's `kind` in the
// reader's language.

import { writable } from 'svelte/store';
import { api } from './api';
import { m } from './paraglide/messages';
import type { InboxItem } from './types';

/** Server default for NOTIFICATION_RETENTION_DAYS, used until /config loads. */
export const DEFAULT_RETENTION_DAYS = 30;

export const inboxItems = writable<InboxItem[]>([]);
export const inboxUnread = writable(0);

const THROTTLE_MS = 30_000;
// Last successful-or-pending load, per phone. Reset on failure so an error
// doesn't block the next attempt for 30s.
let last = { phone: '', ms: 0 };
// The phone of the most recent request: a slower response for a previous phone
// (the user just changed it) must not overwrite the current one.
let current = '';
// Items deleted here, so a load that raced the delete doesn't flash them back.
// id → null while the delete is in flight, then the number of the last load
// started before it completed. Loads started later reflect the delete.
const pendingDeletes = new Map<string, number | null>();
let loadSeq = 0;

function hiddenByPendingDelete(id: string, seq: number): boolean {
	if (!pendingDeletes.has(id)) return false;
	const done = pendingDeletes.get(id);
	return done === null || done === undefined || seq <= done;
}

/**
 * Reload the inbox for phone. Calls for the same phone within 30s of the
 * previous load are skipped unless `force` (used when a push just arrived, the
 * phone changed, or the inbox page opens). Network errors keep the last state.
 */
export async function refreshInbox(phone: string, { force = false } = {}): Promise<void> {
	current = phone;
	if (!phone) {
		inboxItems.set([]);
		inboxUnread.set(0);
		return;
	}
	const now = Date.now();
	if (!force && last.phone === phone && now - last.ms < THROTTLE_MS) return;
	last = { phone, ms: now };
	const seq = ++loadSeq;
	try {
		const res = await api.inbox.list(phone);
		if (phone !== current) return;
		const all = res.items ?? [];
		const items = all.filter((i) => !hiddenByPendingDelete(i.id, seq));
		const hidden = all.filter((i) => hiddenByPendingDelete(i.id, seq) && !i.read).length;
		for (const [id, done] of pendingDeletes) if (done !== null && seq > done) pendingDeletes.delete(id);
		inboxItems.set(items);
		inboxUnread.set(Math.max(0, (res.unread ?? 0) - hidden));
	} catch {
		last = { phone: '', ms: 0 }; // offline / server error: keep state, allow retry
	}
}

/**
 * Mark exactly these items read — the ones the user tapped (in the list, or as
 * a push). The badge drops at once; then the inbox is reloaded so it reflects
 * anything that arrived meanwhile. Best-effort.
 */
export async function markInboxItemsRead(phone: string, ids: string[]): Promise<void> {
	if (!phone || ids.length === 0) return;
	let newlyRead = 0;
	inboxItems.update((list) =>
		list.map((i) => {
			if (!ids.includes(i.id) || i.read) return i;
			newlyRead++;
			return { ...i, read: true };
		})
	);
	if (newlyRead > 0) inboxUnread.update((n) => Math.max(0, n - newlyRead));
	try {
		await api.inbox.markRead(phone, ids);
	} catch {
		return; // the badge just stays until the next successful refresh
	}
	await refreshInbox(phone, { force: true });
}

/**
 * Delete an item: it disappears from the list right away, then the server is
 * told. On failure the list is reloaded so it shows the truth again.
 * Resolves to whether the server accepted it.
 */
export async function deleteInboxItem(phone: string, item: InboxItem): Promise<boolean> {
	if (!phone) return false;
	pendingDeletes.set(item.id, null);
	inboxItems.update((list) => list.filter((i) => i.id !== item.id));
	if (!item.read) inboxUnread.update((n) => Math.max(0, n - 1));
	try {
		await api.inbox.remove(phone, item.id);
	} catch {
		pendingDeletes.delete(item.id);
		await refreshInbox(phone, { force: true });
		return false;
	}
	if (pendingDeletes.has(item.id)) pendingDeletes.set(item.id, loadSeq);
	return true;
}

/** Undo a delete ("Annuler"): restore on the server, then reload the list. */
export async function restoreInboxItem(phone: string, id: string): Promise<boolean> {
	if (!phone) return false;
	pendingDeletes.delete(id);
	let ok = true;
	try {
		await api.inbox.restore(phone, id);
	} catch {
		ok = false; // too late (or offline): the reload shows what the server has
	}
	await refreshInbox(phone, { force: true });
	return ok;
}

/** Query parameter the service worker adds when a push is tapped (sw.js). */
export const PUSH_INBOX_PARAM = 'n';

/**
 * The inbox item id a tapped push opened this page with (`?n=<id>`), and the
 * same URL without it. Pure, for the layout to mark the item read and tidy the
 * address bar.
 */
export function inboxIdFromUrl(url: URL): { id: string | null; clean: URL } {
	const clean = new URL(url.href);
	const id = clean.searchParams.get(PUSH_INBOX_PARAM);
	clean.searchParams.delete(PUSH_INBOX_PARAM);
	return { id: id && id.trim() ? id.trim() : null, clean };
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
	feedback_reminder: '💬',
	engagement_reminder: '📣',
	test: '🔔'
};

/**
 * Localized headline for an item; falls back to the server text for unknown
 * kinds. The test notification's server text is already in the user's language
 * (the quote of the day it was sent with), so it uses that too.
 */
export function inboxTitle(item: InboxItem): string {
	if (item.kind === 'feedback_reminder') return m.inboxKindFeedbackReminder();
	if (item.kind === 'engagement_reminder') return m.inboxKindEngagementReminder();
	const fn = TITLES[item.kind];
	if (fn && item.contact_name) return fn(item.contact_name);
	return item.title || m.inboxTitle();
}

/**
 * "Origin → Destination", or the server body when the route is unknown. The
 * weekly engagement reminder (sent in French to everyone) is about no trip, so
 * its text is rendered in the reader's language instead.
 */
export function inboxRoute(item: InboxItem): string {
	if (item.kind === 'engagement_reminder') return m.inboxEngagementReminderBody();
	if (item.origin && item.destination) return `${item.origin} → ${item.destination}`;
	return item.body;
}

export function inboxIcon(item: InboxItem): string {
	return ICONS[item.kind] ?? '🔔';
}

/**
 * True when the trip this item is about has already departed. Never for the
 * feedback reminder, whose whole point is that the trip is over.
 */
export function isPastTrip(item: InboxItem, now: number = Date.now()): boolean {
	if (!item.departure_at || item.kind === 'feedback_reminder') return false;
	const t = Date.parse(item.departure_at);
	return !Number.isNaN(t) && t < now;
}

/** Bell badge text: hidden at 0, capped at "9+". */
export function badgeText(unread: number): string {
	if (unread <= 0) return '';
	return unread > 9 ? '9+' : String(unread);
}
