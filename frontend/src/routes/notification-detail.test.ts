// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import Detail from './notifications/[id]/+page.svelte';
import { api } from '#lib/api.js';
import { userPhone } from '#lib/stores.js';
import { inboxItems, inboxUnread } from '#lib/inbox.js';
import { m } from '#lib/paraglide/messages.js';
import type { InboxItem } from '#lib/types.js';

const params = vi.hoisted(() => ({ id: 'd1' }));
vi.mock('$app/state', () => ({ page: { params } }));

function item(over: Partial<InboxItem>): InboxItem {
	return {
		id: 'd1', kind: 'morning_digest', contact_name: '', origin: '', destination: '', url: '/',
		title: "☀️ Aujourd'hui sur Go-Stop", body: 'Aujourd’hui : 2 trajets — 08:00 Saillans → Crest · 17:30 Crest → Die.',
		created_at: '2026-10-26T07:00:00Z', read: true,
		...over
	};
}

function serve(items: InboxItem[]) {
	return vi.spyOn(api.inbox, 'list').mockImplementation(async () => ({ items, unread: 0 }));
}

beforeEach(() => {
	localStorage.clear();
	inboxItems.set([]);
	inboxUnread.set(0);
	userPhone.set('0611111111');
});
afterEach(() => {
	vi.restoreAllMocks();
	params.id = 'd1';
	userPhone.set('');
});

describe('notification page', () => {
	it('shows the morning digest in full with a way to the board', async () => {
		serve([item({})]);
		const { container } = render(Detail);
		await waitFor(() => expect(container.querySelector('.notif-body')).not.toBeNull());
		expect(container.textContent).toContain(m.inboxKindMorningDigest());
		expect(container.querySelector('.notif-body')?.textContent).toContain('08:00 Saillans → Crest · 17:30 Crest → Die');
		const action = container.querySelector<HTMLAnchorElement>('.notif-action');
		expect(action?.getAttribute('href')).toBe('/');
		expect(action?.textContent).toContain(m.inboxSeeBoard());
	});

	it('shows a test notification with its quote and no follow-up', async () => {
		params.id = 't1';
		serve([item({ id: 't1', kind: 'test', title: 'Citation du jour', body: 'Un autre monde est possible.' })]);
		const { container } = render(Detail);
		await waitFor(() => expect(container.querySelector('.notif-body')).not.toBeNull());
		expect(container.textContent).toContain('Citation du jour');
		expect(container.querySelector('.notif-body')?.textContent).toContain('Un autre monde est possible.');
		expect(container.textContent).toContain(m.inboxTestTag());
		expect(container.querySelector('.notif-action')).toBeNull();
	});

	it('says so when the notification is gone', async () => {
		params.id = 'gone';
		serve([item({})]);
		const { container } = render(Detail);
		await waitFor(() => expect(container.textContent).toContain(m.inboxNotFound()));
	});
});
