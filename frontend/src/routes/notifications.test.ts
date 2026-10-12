// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';
import Notifications from './notifications/+page.svelte';
import { api } from '$lib/api';
import { userPhone } from '$lib/stores';
import { inboxUnread } from '$lib/inbox';
import { pushState } from '$lib/pwa';
import { m } from '$lib/paraglide/messages';
import type { InboxItem } from '$lib/types';

const items: InboxItem[] = [
	{
		id: 'n2', kind: 'interest_received', contact_name: 'Bob', origin: 'Saillans', destination: 'Die',
		url: '/my-rides', title: 'x', body: 'y', created_at: '2026-10-12T10:00:00Z', read: false
	},
	{
		id: 'n1', kind: 'ride_match', contact_name: 'Alice', origin: 'Saillans', destination: 'Crest',
		departure_at: '2020-01-01T08:00:00Z', url: '/rides/r1', title: 'x', body: 'y',
		created_at: '2026-10-11T10:00:00Z', read: true
	}
];

beforeEach(() => {
	localStorage.clear();
	userPhone.set('');
	inboxUnread.set(0);
	pushState.set('subscribed');
});
afterEach(() => vi.restoreAllMocks());

describe('notifications page', () => {
	it('lists items newest first, highlights unread, then marks all read', async () => {
		userPhone.set('0611111111');
		vi.spyOn(api.inbox, 'list').mockResolvedValue({ items, unread: 1 });
		const mark = vi.spyOn(api.inbox, 'markRead').mockResolvedValue(null);

		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));

		const [first, second] = Array.from(container.querySelectorAll<HTMLAnchorElement>('.inbox-item'));
		expect(first.getAttribute('href')).toBe('/my-rides');
		expect(first.textContent).toContain(m.inboxKindInterestReceived({ name: 'Bob' }));
		expect(first.classList.contains('inbox-unread')).toBe(true);
		expect(second.classList.contains('inbox-unread')).toBe(false);
		expect(second.textContent).toContain('Saillans → Crest');
		expect(second.querySelector('.tag')?.textContent).toBe(m.inboxPast());

		await waitFor(() => expect(mark).toHaveBeenCalledWith('0611111111'));
		await waitFor(() => expect(get(inboxUnread)).toBe(0));
	});

	it('does not call mark-read when everything is already read', async () => {
		userPhone.set('0611111111');
		vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [items[1]], unread: 0 });
		const mark = vi.spyOn(api.inbox, 'markRead').mockResolvedValue(null);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		expect(mark).not.toHaveBeenCalled();
	});

	it('shows an empty state', async () => {
		userPhone.set('0611111111');
		vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [], unread: 0 });
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelector('.empty')?.textContent).toBe(m.inboxEmpty()));
	});

	it('warns when push is off on this device', async () => {
		userPhone.set('0611111111');
		pushState.set('default');
		vi.spyOn(api.inbox, 'list').mockResolvedValue({ items: [], unread: 0 });
		const { container } = render(Notifications);
		expect(container.querySelector('.inbox-push-off')).not.toBeNull();
	});

	it('asks for a phone when the profile has none', () => {
		const list = vi.spyOn(api.inbox, 'list');
		const { container } = render(Notifications);
		expect(container.querySelector('.inbox-no-phone')).not.toBeNull();
		expect(list).not.toHaveBeenCalled();
	});
});
