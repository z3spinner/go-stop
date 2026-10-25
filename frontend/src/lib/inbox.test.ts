// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { api } from './api';
import {
	badgeText,
	inboxIcon,
	inboxIdFromUrl,
	inboxItems,
	inboxRoute,
	inboxTitle,
	inboxUnread,
	isPastTrip,
	markInboxItemsRead,
	refreshInbox,
	deleteInboxItem,
	restoreInboxItem
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
	it('renders the weekly engagement reminder in the reader\'s language', () => {
		const weekly = item({
			kind: 'engagement_reminder',
			contact_name: '',
			origin: '',
			destination: '',
			title: 'Proposez un trajet cette semaine ! 🚗',
			body: 'Le tableau est presque vide…'
		});
		expect(inboxTitle(weekly)).toBe(m.inboxKindEngagementReminder());
		expect(inboxRoute(weekly)).toBe(m.inboxEngagementReminderBody());
		expect(inboxIcon(weekly)).toBe('📣');
	});
	it('renders the alert expiry warning in the reader\'s language, with its route', () => {
		const expiring = item({
			kind: 'alert_expiring',
			contact_name: '',
			origin: 'Crest',
			destination: 'Saillans',
			title: 'Votre alerte expire bientôt'
		});
		expect(inboxTitle(expiring)).toBe(m.inboxKindAlertExpiring());
		expect(inboxRoute(expiring)).toBe('Crest → Saillans');
		expect(inboxIcon(expiring)).toBe('⏳');
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
	it('shows a test notification with its own quote text', () => {
		const t = item({ kind: 'test', contact_name: '', origin: '', destination: '', title: 'Citation du jour', body: 'Un autre monde est possible.' });
		expect(inboxTitle(t)).toBe('Citation du jour');
		expect(inboxRoute(t)).toBe('Un autre monde est possible.');
		expect(inboxIcon(t)).toBe('🔔');
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
	it('never tags the feedback reminder (the trip being over is its point)', () => {
		expect(isPastTrip(item({ kind: 'feedback_reminder', departure_at: '2026-10-12T11:00:00Z' }), now)).toBe(false);
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

	it('throttles per phone: another phone is not throttled', async () => {
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [], unread: 0 });
		await refreshInbox('0611111111', { force: true });
		await refreshInbox('0622222222');
		expect(list).toHaveBeenCalledTimes(2);
	});

	it('a failed load does not block the next attempt', async () => {
		const list = vi.spyOn(api.inbox, 'list').mockRejectedValueOnce(new Error('offline'));
		list.mockResolvedValue({ items: [item()], unread: 1 });
		await refreshInbox('0633333333', { force: true });
		await refreshInbox('0633333333'); // not forced, right after the failure
		expect(list).toHaveBeenCalledTimes(2);
		expect(get(inboxUnread)).toBe(1);
	});

	it('ignores a late response for a phone that was replaced', async () => {
		let resolveOld!: (v: { items: InboxItem[]; unread: number }) => void;
		const list = vi.spyOn(api.inbox, 'list');
		list.mockImplementationOnce(() => new Promise((r) => (resolveOld = r)));
		list.mockResolvedValueOnce({ items: [item({ id: 'new' })], unread: 1 });
		const old = refreshInbox('0644444444', { force: true });
		await refreshInbox('0655555555', { force: true });
		resolveOld({ items: [item({ id: 'old' })], unread: 5 });
		await old;
		expect(get(inboxItems).map((i) => i.id)).toEqual(['new']);
		expect(get(inboxUnread)).toBe(1);
	});

	it('marks exactly the given ids read, then reloads', async () => {
		const mark = vi.spyOn(api.inbox, 'markRead').mockResolvedValue(null);
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [item({ read: true })], unread: 0 });
		inboxUnread.set(1);
		await markInboxItemsRead('0611111111', ['n1']);
		expect(mark).toHaveBeenCalledWith('0611111111', ['n1']);
		expect(list).toHaveBeenCalledWith('0611111111');
		expect(get(inboxUnread)).toBe(0);
	});

	it('drops the badge before the server answers', async () => {
		inboxItems.set([item({ id: 'q1', read: false }), item({ id: 'q2', read: false })]);
		inboxUnread.set(2);
		let finish!: () => void;
		vi.spyOn(api.inbox, 'markRead').mockImplementation(() => new Promise((r) => (finish = () => r(null))));
		vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [], unread: 1 });
		const p = markInboxItemsRead('0612121212', ['q1']);
		expect(get(inboxUnread)).toBe(1);
		expect(get(inboxItems).find((i) => i.id === 'q1')?.read).toBe(true);
		expect(get(inboxItems).find((i) => i.id === 'q2')?.read).toBe(false);
		finish();
		await p;
	});

	it('does nothing for an empty id list', async () => {
		const mark = vi.spyOn(api.inbox, 'markRead');
		await markInboxItemsRead('0611111111', []);
		expect(mark).not.toHaveBeenCalled();
	});

	it('delete removes the item at once, then tells the server', async () => {
		inboxItems.set([item({ id: 'a', read: true }), item({ id: 'b', read: false })]);
		inboxUnread.set(1);
		const remove = vi.spyOn(api.inbox, 'remove').mockResolvedValue(null);
		const ok = await deleteInboxItem('0611111111', item({ id: 'b', read: false }));
		expect(ok).toBe(true);
		expect(remove).toHaveBeenCalledWith('0611111111', 'b');
		expect(get(inboxItems).map((i) => i.id)).toEqual(['a']);
		expect(get(inboxUnread)).toBe(0);
	});

	it('a load that raced the delete does not bring the item back', async () => {
		inboxItems.set([item({ id: 'gone' })]);
		let finishDelete!: () => void;
		vi.spyOn(api.inbox, 'remove').mockImplementation(() => new Promise((r) => (finishDelete = () => r(null))));
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [item({ id: 'gone' }), item({ id: 'kept' })], unread: 2 });
		const del = deleteInboxItem('0677777777', item({ id: 'gone' }));
		await refreshInbox('0677777777', { force: true }); // answered before the delete landed
		expect(get(inboxItems).map((i) => i.id)).toEqual(['kept']);
		expect(get(inboxUnread)).toBe(1);
		finishDelete();
		await del;
		// A load started after the delete reflects the server again.
		list.mockResolvedValue({ items: [item({ id: 'kept' })], unread: 1 });
		await refreshInbox('0677777777', { force: true });
		expect(get(inboxItems).map((i) => i.id)).toEqual(['kept']);
	});

	it('a failed delete reloads the truth', async () => {
		inboxItems.set([item({ id: 'x1' })]);
		vi.spyOn(api.inbox, 'remove').mockRejectedValue(new Error('offline'));
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [item({ id: 'x1' })], unread: 1 });
		const ok = await deleteInboxItem('0688888888', item({ id: 'x1' }));
		expect(ok).toBe(false);
		expect(list).toHaveBeenCalled();
		expect(get(inboxItems).map((i) => i.id)).toEqual(['x1']);
	});

	it('undo restores on the server and reloads', async () => {
		const restore = vi.spyOn(api.inbox, 'restore').mockResolvedValue(null);
		vi.spyOn(api.inbox, 'remove').mockResolvedValue(null);
		const list = vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [item({ id: 'u1', read: true })], unread: 0 });
		await deleteInboxItem('0699999999', item({ id: 'u1' }));
		expect(await restoreInboxItem('0699999999', 'u1')).toBe(true);
		expect(restore).toHaveBeenCalledWith('0699999999', 'u1');
		expect(list).toHaveBeenCalledWith('0699999999');
		expect(get(inboxItems).map((i) => i.id)).toEqual(['u1']); // no longer hidden as pending
	});
});

describe('inboxIdFromUrl', () => {
	it('reads the tapped push\'s inbox id and strips it, keeping other params', () => {
		const { id, clean } = inboxIdFromUrl(new URL('https://x.test/my-rides?tab=2&n=abc-123#top'));
		expect(id).toBe('abc-123');
		expect(clean.href).toBe('https://x.test/my-rides?tab=2#top');
	});
	it('is null without the parameter (or with an empty one)', () => {
		expect(inboxIdFromUrl(new URL('https://x.test/my-rides')).id).toBeNull();
		expect(inboxIdFromUrl(new URL('https://x.test/my-rides?n=')).id).toBeNull();
	});
});
