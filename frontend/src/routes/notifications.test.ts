// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';
import Notifications from './notifications/+page.svelte';
import { api } from '$lib/api';
import { userPhone } from '$lib/stores';
import { inboxItems, inboxUnread, refreshInbox } from '$lib/inbox';
import { pushState } from '$lib/pwa';
import { m } from '$lib/paraglide/messages';
import type { InboxItem } from '$lib/types';

function item(over: Partial<InboxItem>): InboxItem {
	return {
		id: 'x', kind: 'ride_match', contact_name: 'Alice', origin: 'Saillans', destination: 'Crest',
		url: '/rides/r1', title: 'x', body: 'y', created_at: '2026-10-12T10:00:00Z', read: false,
		...over
	};
}

const unreadBob = item({ id: 'n2', kind: 'interest_received', contact_name: 'Bob', destination: 'Die', url: '/my-rides' });
const readPast = item({ id: 'n1', departure_at: '2020-01-01T08:00:00Z', created_at: '2026-10-11T10:00:00Z', read: true });

/** Server double: list returns `server`; markRead flips those ids to read. */
function fakeServer(initial: InboxItem[]) {
	let server = initial.map((i) => ({ ...i }));
	const list = vi.spyOn(api.inbox, 'list').mockImplementation(async () => ({
		items: server.map((i) => ({ ...i })),
		unread: server.filter((i) => !i.read).length
	}));
	const markRead = vi.spyOn(api.inbox, 'markRead').mockImplementation(async (_p, ids) => {
		server = server.map((i) => (ids.includes(i.id) ? { ...i, read: true } : i));
		return null;
	});
	const markAll = vi.spyOn(api.inbox, 'markAllRead');
	return {
		list,
		markRead,
		markAll,
		push(n: InboxItem) {
			server = [n, ...server];
		}
	};
}

beforeEach(() => {
	localStorage.clear();
	userPhone.set('');
	inboxItems.set([]);
	inboxUnread.set(0);
	pushState.set('subscribed');
});
afterEach(() => vi.restoreAllMocks());

describe('notifications page', () => {
	it('lists items, highlights unread, and marks read only the ids shown', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([unreadBob, readPast]);

		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));

		const [first, second] = Array.from(container.querySelectorAll<HTMLAnchorElement>('.inbox-item'));
		expect(first.getAttribute('href')).toBe('/my-rides');
		expect(first.textContent).toContain(m.inboxKindInterestReceived({ name: 'Bob' }));
		expect(first.querySelector('.sr-only')?.textContent).toContain(m.inboxUnreadLabel());
		expect(second.classList.contains('inbox-unread')).toBe(false);
		expect(second.querySelector('.sr-only')).toBeNull();
		expect(second.textContent).toContain('Saillans → Crest');
		expect(second.querySelector('.tag')?.textContent).toBe(m.inboxPast());

		await waitFor(() => expect(srv.markRead).toHaveBeenCalledWith('0611111111', ['n2']));
		expect(srv.markAll).not.toHaveBeenCalled();
		await waitFor(() => expect(get(inboxUnread)).toBe(0));
		// Still highlighted for this visit after the server marked it read.
		expect(container.querySelector('.inbox-item')?.classList.contains('inbox-unread')).toBe(true);
	});

	it('shows a notification that arrives while open, then marks just that one', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		expect(srv.markRead).not.toHaveBeenCalled();

		srv.push(item({ id: 'n3', contact_name: 'Carla' }));
		await refreshInbox('0611111111', { force: true }); // what the layout does on 'inbox-updated'

		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));
		expect(container.querySelector('.inbox-item')?.textContent).toContain(m.inboxKindRideMatch({ name: 'Carla' }));
		await waitFor(() => expect(srv.markRead).toHaveBeenCalledWith('0611111111', ['n3']));
		expect(srv.markRead).toHaveBeenCalledTimes(1);
	});

	it('does not call mark-read when everything is already read', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		expect(srv.markRead).not.toHaveBeenCalled();
	});

	it('shows an empty state', async () => {
		userPhone.set('0611111111');
		fakeServer([]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelector('.empty')?.textContent).toBe(m.inboxEmpty()));
	});

	it('warns when push is off on this device', () => {
		userPhone.set('0611111111');
		pushState.set('default');
		fakeServer([]);
		const { container } = render(Notifications);
		expect(container.querySelector('.inbox-push-off')).not.toBeNull();
	});

	it('asks for a phone, then loads once one is entered', async () => {
		const srv = fakeServer([unreadBob]);
		const { container } = render(Notifications);
		expect(container.querySelector('.inbox-no-phone')).not.toBeNull();
		expect(srv.list).not.toHaveBeenCalled();

		userPhone.set('0611111111');
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		expect(container.querySelector('.inbox-no-phone')).toBeNull();
	});
});
