// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { api } from './api';
import {
	badgeText,
	inboxIcon,
	inboxItems,
	inboxRoute,
	inboxTitle,
	inboxUnread,
	isPastTrip,
	markAllInboxRead,
	refreshInbox
} from './inbox';
import { m } from './paraglide/messages';
import type { InboxItem } from './types';

function item(over: Partial<InboxItem> = {}): InboxItem {
	return {
		id: 'n1',
		kind: 'ride_match',
		contact_name: 'Alice',
		origin: 'Saillans',
		destination: 'Crest',
		url: '/rides/r1',
		title: 'Ride available!',
		body: 'Alice is driving from Saillans to Crest',
		created_at: '2026-10-12T10:00:00Z',
		read: false,
		...over
	};
}

afterEach(() => vi.restoreAllMocks());

describe('inboxTitle', () => {
	it('renders known kinds from the contact name in the active locale', () => {
		expect(inboxTitle(item())).toBe(m.inboxKindRideMatch({ name: 'Alice' }));
		expect(inboxTitle(item({ kind: 'contact_offered' }))).toBe(m.inboxKindContactOffered({ name: 'Alice' }));
	});
	it('needs no name for the feedback reminder', () => {
		expect(inboxTitle(item({ kind: 'feedback_reminder', contact_name: '' }))).toBe(m.inboxKindFeedbackReminder());
	});
	it('falls back to the server title for unknown kinds or a missing name', () => {
		expect(inboxTitle(item({ kind: 'something_new' }))).toBe('Ride available!');
		expect(inboxTitle(item({ contact_name: '' }))).toBe('Ride available!');
	});
});

describe('inboxRoute / inboxIcon', () => {
	it('shows the route, else the server body', () => {
		expect(inboxRoute(item())).toBe('Saillans → Crest');
		expect(inboxRoute(item({ origin: '' }))).toBe('Alice is driving from Saillans to Crest');
	});
	it('has a generic icon for unknown kinds', () => {
		expect(inboxIcon(item({ kind: 'something_new' }))).toBe('🔔');
	});
});

describe('isPastTrip', () => {
	const now = Date.parse('2026-10-12T12:00:00Z');
	it('is false without a departure', () => {
		expect(isPastTrip(item(), now)).toBe(false);
	});
	it('compares the departure to now', () => {
		expect(isPastTrip(item({ departure_at: '2026-10-12T11:00:00Z' }), now)).toBe(true);
		expect(isPastTrip(item({ departure_at: '2026-10-12T13:00:00Z' }), now)).toBe(false);
	});
});

describe('badgeText', () => {
	it('hides at zero and caps at 9+', () => {
		expect(badgeText(0)).toBe('');
		expect(badgeText(3)).toBe('3');
		expect(badgeText(12)).toBe('9+');
	});
});

describe('refreshInbox / markAllInboxRead', () => {
	beforeEach(() => {
		inboxItems.set([]);
		inboxUnread.set(0);
	});

	it('loads items and the unread count', async () => {
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [item()], unread: 1 });
		await refreshInbox('0611111111', { force: true });
		expect(list).toHaveBeenCalledWith('0611111111');
		expect(get(inboxItems)).toHaveLength(1);
		expect(get(inboxUnread)).toBe(1);
	});

	it('throttles non-forced reloads', async () => {
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [], unread: 0 });
		await refreshInbox('0611111111', { force: true });
		await refreshInbox('0611111111');
		expect(list).toHaveBeenCalledTimes(1);
	});

	it('keeps the last state on network errors', async () => {
		inboxUnread.set(2);
		vi.spyOn(api.inbox, 'list').mockRejectedValue(new Error('offline'));
		await refreshInbox('0611111111', { force: true });
		expect(get(inboxUnread)).toBe(2);
	});

	it('clears everything without a phone', async () => {
		inboxUnread.set(2);
		const list = vi.spyOn(api.inbox, 'list');
		await refreshInbox('');
		expect(list).not.toHaveBeenCalled();
		expect(get(inboxUnread)).toBe(0);
	});

	it('mark-all-read clears the badge', async () => {
		inboxUnread.set(4);
		const mark = vi.spyOn(api.inbox, 'markRead').mockResolvedValue(null);
		await markAllInboxRead('0611111111');
		expect(mark).toHaveBeenCalledWith('0611111111');
		expect(get(inboxUnread)).toBe(0);
	});
});
