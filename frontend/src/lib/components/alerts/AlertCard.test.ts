// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AlertCard from './AlertCard.svelte';
import { userPhone } from '#lib/stores.js';
import { m } from '#lib/paraglide/messages.js';
import { formatDate } from '#lib/utils.js';
import type { Request } from '#lib/types.js';

const ZERO = '0001-01-01T00:00:00Z';
const DAY = 24 * 60 * 60 * 1000;
const inDays = (d: number) => new Date(Date.now() + d * DAY).toISOString();

const alert = (over: Partial<Request> = {}): Request => ({
	ID: 'req1',
	SearcherName: 'Zoé',
	Phone: '0622000002',
	Origin: 'Crest',
	Destination: 'Saillans',
	Date: ZERO,
	DepartureAt: ZERO,
	Flexibility: 0,
	PostedAt: inDays(-27),
	ExpiresAt: inDays(2),
	...over
});

// Answers the card's own calls: its contact offers (GET) and a renewal (POST).
function stubApi(renewed: string) {
	const fetchMock = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
		if (String(url).endsWith('/renew') && init?.method === 'POST') {
			return new Response(JSON.stringify(alert({ ExpiresAt: renewed })), { status: 200 });
		}
		return new Response('[]', { status: 200 });
	});
	vi.stubGlobal('fetch', fetchMock);
	return fetchMock;
}

beforeEach(() => userPhone.set('0622000002'));
afterEach(() => vi.unstubAllGlobals());

describe('AlertCard expiry', () => {
	it('shows when an open-ended alert expires and offers to renew it in its last week', () => {
		stubApi(inDays(30));
		const { container } = render(AlertCard, { props: { request: alert() } });
		const tag = container.querySelector('.tag-expires');
		expect(tag?.textContent).toBe(m.alertExpiresOn({ date: formatDate(alert().ExpiresAt) }));
		expect(tag?.classList.contains('tag-expiring')).toBe(true);
		expect(container.querySelector('.btn-renew')).not.toBeNull();
	});

	it('offers no renewal while there is more than a week left', () => {
		stubApi(inDays(30));
		const { container } = render(AlertCard, { props: { request: alert({ ExpiresAt: inDays(20) }) } });
		expect(container.querySelector('.tag-expires')).not.toBeNull();
		expect(container.querySelector('.btn-renew')).toBeNull();
	});

	it('shows no expiry for a dated request (it expires with its date)', () => {
		stubApi(inDays(30));
		const { container } = render(AlertCard, {
			props: { request: alert({ Date: '2030-06-01T00:00:00Z', DepartureAt: '2030-06-01T09:00:00Z' }) }
		});
		expect(container.querySelector('.tag-expires')).toBeNull();
		expect(container.querySelector('.btn-renew')).toBeNull();
	});

	it('renewing POSTs with the owner phone and shows the new expiry', async () => {
		const renewed = inDays(30);
		const fetchMock = stubApi(renewed);
		const { container } = render(AlertCard, { props: { request: alert() } });

		await fireEvent.click(container.querySelector('.btn-renew')!);

		await vi.waitFor(() =>
			expect(container.querySelector('.delete-msg')?.textContent).toBe(m.alertRenewed({ date: formatDate(renewed) }))
		);
		const call = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/renew'))!;
		expect(String(call[0])).toBe('/api/requests/req1/renew');
		expect(new Headers(call[1]?.headers).get('X-Phone')).toBe('0622000002');
		expect(container.querySelector('.btn-renew')).toBeNull(); // no longer expiring soon
	});
});
