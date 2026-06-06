// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import Search from './search/+page.svelte';
import { userPhone } from '$lib/stores';

afterEach(() => vi.unstubAllGlobals());

vi.mock('$app/state', () => ({
	page: { url: new URL('http://localhost/search?origin=Saillans&destination=Crest') }
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

describe('search', () => {
	it('renders two result columns after an auto-submitted query', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Search);
		await vi.waitFor(() => expect(container.querySelectorAll('.results-col-header').length).toBe(2));
	});

	it('marks the user\'s own ride in results with a badge and manage link', async () => {
		userPhone.set('0612345678');
		const ownRide = {
			ID: '42', DriverName: 'Me', Origin: 'Saillans', Destination: 'Crest',
			DepartureAt: '2030-06-15T08:00:00Z', Flexibility: 0, InterestCount: 0
		};
		vi.stubGlobal('fetch', vi.fn(async (url: RequestInfo | URL) => {
			const u = String(url);
			if (u.includes('/api/rides')) return new Response(JSON.stringify([ownRide]), { status: 200 });
			return new Response('[]', { status: 200 }); // /api/interests, /api/destinations, etc.
		}));
		const { container } = render(Search);
		await vi.waitFor(() => {
			expect(container.querySelector('.tag-your-ride')).toBeTruthy();
			expect(container.querySelector('a.ride-manage-link')!.getAttribute('href')).toBe('/my-rides#card-42');
		});
		userPhone.set('');
	});
});
