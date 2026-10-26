// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import Notifications from './notifications/+page.svelte';
import { api } from '$lib/api';
import { userPhone } from '$lib/stores';
import { inboxItems, inboxUnread, refreshInbox } from '$lib/inbox';
import { pushState } from '$lib/pwa';
import { config } from '$lib/config';
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

/** Server double: list returns `server`; markRead flips those ids to read; remove/restore hide/unhide. */
function fakeServer(initial: InboxItem[]) {
	let server = initial.map((i) => ({ ...i }));
	const dismissed = new Set<string>();
	const visible = () => server.filter((i) => !dismissed.has(i.id));
	const list = vi.spyOn(api.inbox, 'list').mockImplementation(async () => ({
		items: visible().map((i) => ({ ...i })),
		unread: visible().filter((i) => !i.read).length
	}));
	const markRead = vi.spyOn(api.inbox, 'markRead').mockImplementation(async (_p, ids) => {
		server = server.map((i) => (ids.includes(i.id) ? { ...i, read: true } : i));
		return null;
	});
	const markAll = vi.spyOn(api.inbox, 'markAllRead');
	const remove = vi.spyOn(api.inbox, 'remove').mockImplementation(async (_p, id) => {
		dismissed.add(id);
		server = server.map((i) => (i.id === id ? { ...i, read: true } : i));
		return null;
	});
	const restore = vi.spyOn(api.inbox, 'restore').mockImplementation(async (_p, id) => {
		dismissed.delete(id);
		return null;
	});
	return {
		list,
		markRead,
		markAll,
		remove,
		restore,
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
	it('lists items and highlights unread ones without marking them read', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([unreadBob, readPast]);

		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));

		const [first, second] = Array.from(container.querySelectorAll<HTMLAnchorElement>('.inbox-item'));
		expect(first.getAttribute('href')).toBe('/my-rides');
		expect(first.textContent).toContain(m.inboxKindInterestReceived({ name: 'Bob' }));
		expect(first.classList.contains('inbox-unread')).toBe(true);
		expect(first.querySelector('.sr-only')?.textContent).toContain(m.inboxUnreadLabel());
		expect(second.classList.contains('inbox-unread')).toBe(false);
		expect(second.querySelector('.sr-only')).toBeNull();
		expect(second.textContent).toContain('Saillans → Crest');
		expect(second.querySelector('.tag')?.textContent).toBe(m.inboxPast());

		// Being on screen is not reading: the count (the bell's badge) stays.
		await new Promise((r) => setTimeout(r, 50));
		expect(srv.markRead).not.toHaveBeenCalled();
		expect(srv.markAll).not.toHaveBeenCalled();
		expect(get(inboxUnread)).toBe(1);
	});

	it('tapping a notification marks just that one read', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([unreadBob, item({ id: 'n4', contact_name: 'Dana' }), readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(3));
		expect(get(inboxUnread)).toBe(2);
		container.addEventListener('click', (e) => e.preventDefault()); // jsdom can't navigate

		await fireEvent.click(container.querySelector('[data-id="n2"]')!);
		expect(get(inboxUnread)).toBe(1); // the badge drops at once
		await waitFor(() => expect(srv.markRead).toHaveBeenCalledWith('0611111111', ['n2']));
		expect(container.querySelector('[data-id="n4"]')?.classList.contains('inbox-unread')).toBe(true);

		// Tapping an already-read one doesn't call the server.
		await fireEvent.click(container.querySelector('[data-id="n1"]')!);
		expect(srv.markRead).toHaveBeenCalledTimes(1);
	});

	it('a notification arriving while open is shown and stays unread', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));

		srv.push(item({ id: 'n3', contact_name: 'Carla' }));
		await refreshInbox('0611111111', { force: true }); // what the layout does on 'inbox-updated'

		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));
		expect(container.querySelector('.inbox-item')?.textContent).toContain(m.inboxKindRideMatch({ name: 'Carla' }));
		expect(container.querySelector('.inbox-item')?.classList.contains('inbox-unread')).toBe(true);
		expect(srv.markRead).not.toHaveBeenCalled();
		expect(get(inboxUnread)).toBe(1);
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

	it('labels a test notification as such', async () => {
		userPhone.set('0611111111');
		fakeServer([item({ id: 't1', kind: 'test', contact_name: '', origin: '', destination: '', url: '/', title: 'Citation du jour', body: 'Un autre monde est possible.', read: true })]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		const el = container.querySelector('.inbox-item')!;
		expect(el.textContent).toContain('Citation du jour');
		expect(el.textContent).toContain('Un autre monde est possible.');
		expect(el.querySelector('.inbox-test-tag')?.textContent).toBe(m.inboxTestTag());
	});

	it('deletes with the delete button, then undo brings it back', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([unreadBob, readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));

		const btn = container.querySelector<HTMLButtonElement>('.swipe-delete-btn')!;
		expect(btn.getAttribute('aria-label')).toBe(m.inboxDeleteAria());
		await fireEvent.click(btn);

		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		expect(srv.remove).toHaveBeenCalledWith('0611111111', 'n2');
		expect(container.querySelector('[data-id="n2"]')).toBeNull();
		const undo = container.querySelector<HTMLButtonElement>('.btn-inbox-undo')!;
		expect(container.querySelector('.inbox-undo')?.textContent).toContain(m.inboxDeleted());

		await fireEvent.click(undo);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(2));
		// Deleting made it read; it comes back read.
		expect(container.querySelector('[data-id="n2"]')?.classList.contains('inbox-unread')).toBe(false);
		expect(srv.restore).toHaveBeenCalledWith('0611111111', 'n2');
		expect(container.querySelector('.inbox-undo')).toBeNull();
	});

	it('the Delete key deletes the focused notification', async () => {
		userPhone.set('0611111111');
		const srv = fakeServer([readPast]);
		const { container } = render(Notifications);
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
		await fireEvent.keyDown(container.querySelector('.inbox-item')!, { key: 'Delete' });
		await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(0));
		expect(srv.remove).toHaveBeenCalledWith('0611111111', 'n1');
	});

	describe('swipe', () => {
		function pointer(el: Element, type: string, x: number, y = 100) {
			const e = new MouseEvent(type, { clientX: x, clientY: y, bubbles: true, cancelable: true, button: 0 });
			Object.defineProperty(e, 'pointerId', { value: 1 });
			Object.defineProperty(e, 'pointerType', { value: 'touch' });
			el.dispatchEvent(e);
		}
		let now = 0;
		beforeEach(() => {
			now = 0;
			vi.spyOn(performance, 'now').mockImplementation(() => now);
		});

		it('a quick horizontal swipe deletes the notification', async () => {
			userPhone.set('0611111111');
			const srv = fakeServer([readPast]);
			const { container } = render(Notifications);
			await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
			const row = container.querySelector('.swipe-front')!;
			pointer(row, 'pointerdown', 300);
			pointer(row, 'pointermove', 280);
			pointer(row, 'pointermove', 150);
			now = 120; // 150px in 120ms: a flick
			pointer(row, 'pointerup', 150);
			await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(0));
			expect(srv.remove).toHaveBeenCalledWith('0611111111', 'n1');
		});

		it('a short slow drag springs back', async () => {
			userPhone.set('0611111111');
			const srv = fakeServer([readPast]);
			const { container } = render(Notifications);
			await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
			const row = container.querySelector<HTMLElement>('.swipe-front')!;
			pointer(row, 'pointerdown', 300);
			pointer(row, 'pointermove', 270);
			now = 1000;
			pointer(row, 'pointerup', 270);
			await new Promise((r) => setTimeout(r, 400));
			expect(container.querySelectorAll('.inbox-item')).toHaveLength(1);
			expect(row.style.transform).toBe('translateX(0px)');
			expect(srv.remove).not.toHaveBeenCalled();
		});

		it('vertical movement is a scroll, not a swipe', async () => {
			userPhone.set('0611111111');
			const srv = fakeServer([readPast]);
			const { container } = render(Notifications);
			await waitFor(() => expect(container.querySelectorAll('.inbox-item')).toHaveLength(1));
			const row = container.querySelector<HTMLElement>('.swipe-front')!;
			pointer(row, 'pointerdown', 300, 100);
			pointer(row, 'pointermove', 290, 220);
			pointer(row, 'pointermove', 100, 240); // sideways later doesn't turn it into a swipe
			now = 100;
			pointer(row, 'pointerup', 100, 240);
			await new Promise((r) => setTimeout(r, 400));
			expect(container.querySelectorAll('.inbox-item')).toHaveLength(1);
			expect(srv.remove).not.toHaveBeenCalled();
		});
	});
});

describe('morning digest switch', () => {
	beforeEach(() => config.set({ siteName: 'Go-Stop', returnDelayHours: 2, morningDigestHour: 8 }));
	afterEach(() => config.set({ siteName: 'Go-Stop', returnDelayHours: 2 }));

	it('shows the saved choice and saves a change', async () => {
		userPhone.set('0611111111');
		fakeServer([]);
		vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });
		const set = vi.spyOn(api.preferences, 'set').mockImplementation(async (_p, prefs) => prefs);

		const { container } = render(Notifications);
		const toggle = await waitFor(() => {
			const el = container.querySelector<HTMLInputElement>('.digest-toggle');
			expect(el).not.toBeNull();
			return el!;
		});
		expect(toggle.checked).toBe(true);
		expect(container.querySelector('.digest-pref')?.textContent).toContain(m.digestToggleHint({ hour: '8' }));

		await fireEvent.click(toggle);
		await waitFor(() => expect(set).toHaveBeenCalledWith('0611111111', { morning_digest: false }));
		expect(toggle.checked).toBe(false);
	});

	it('goes back and says so when saving fails', async () => {
		userPhone.set('0611111111');
		fakeServer([]);
		vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });
		vi.spyOn(api.preferences, 'set').mockRejectedValue(new Error('offline'));

		const { container } = render(Notifications);
		const toggle = await waitFor(() => {
			const el = container.querySelector<HTMLInputElement>('.digest-toggle');
			expect(el).not.toBeNull();
			return el!;
		});
		await fireEvent.click(toggle);
		await waitFor(() => expect(container.querySelector('.digest-err')?.textContent).toBe(m.digestSaveErr()));
		expect(toggle.checked).toBe(true);
	});

	it('is hidden when the server sends no digest', async () => {
		config.set({ siteName: 'Go-Stop', returnDelayHours: 2, morningDigestHour: -1 });
		userPhone.set('0611111111');
		const srv = fakeServer([]);
		vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });

		const { container } = render(Notifications);
		await waitFor(() => expect(srv.list).toHaveBeenCalled());
		expect(container.querySelector('.digest-toggle')).toBeNull();
	});
});
