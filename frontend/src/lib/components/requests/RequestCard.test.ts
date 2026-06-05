// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import RequestCard from './RequestCard.svelte';

const base = { id: 'int7', ride_id: 'r1', driver_name: 'Alice', origin: 'Saillans', destination: 'Crest', departure_at: '2030-06-15T08:00:00Z' };

beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());

describe('RequestCard', () => {
	it('pending shows the waiting label, no contact link', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'pending' } } });
		expect(container.querySelector('.btn-contact-link')).toBeNull();
	});
	it('accepted shows a contact link to /interests/<id>', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'accepted' } } });
		const a = container.querySelector('.btn-contact-link') as HTMLAnchorElement;
		expect(a.getAttribute('href')).toBe('/interests/int7');
	});
	it('accepted shows no cancel button', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'accepted' }, phone: '0622000002' } });
		expect(container.querySelector('.btn-interest-cancel')).toBeNull();
	});
	it('cancelling a pending request DELETEs it and calls oncancelled', async () => {
		localStorage.setItem('interest_r1', 'int7');
		const fetchMock = vi.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		vi.stubGlobal('fetch', fetchMock);
		let cancelledId = '';
		const { container } = render(RequestCard, {
			props: { interest: { ...base, status: 'pending' }, phone: '0622000002', oncancelled: (id: string) => (cancelledId = id) }
		});

		await fireEvent.click(container.querySelector('.btn-interest-cancel')!);

		await vi.waitFor(() => expect(cancelledId).toBe('int7'));
		expect(localStorage.getItem('interest_r1')).toBeNull();
		const [url, init] = fetchMock.mock.calls[0];
		expect(String(url)).toContain('/api/interests/int7');
		expect(init?.method).toBe('DELETE');
	});
});
