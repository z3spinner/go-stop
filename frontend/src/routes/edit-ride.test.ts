// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Edit from './rides/[id]/edit/+page.svelte';
import { userPhone } from '$lib/stores';

const goto = vi.fn();
vi.mock('$app/state', () => ({ page: { params: { id: 'ride-1' } } }));
vi.mock('$app/navigation', () => ({ goto: (...a: unknown[]) => goto(...a) }));

afterEach(() => {
	vi.unstubAllGlobals();
	goto.mockClear();
	userPhone.set('');
	localStorage.clear();
});

describe('edit ride page', () => {
	// The page loads the ride, prefills the form, and PUTs the (possibly changed)
	// fields with the owner's phone, then returns to my-rides.
	it('prefills from the ride and PUTs the changes', async () => {
		userPhone.set('0612345678');
		const calls: { method: string; url: string; body?: string }[] = [];
		vi.stubGlobal(
			'fetch',
			vi.fn(async (url: unknown, init?: RequestInit) => {
				const u = String(url);
				calls.push({ method: init?.method ?? 'GET', url: u, body: init?.body as string });
				if (u.includes('/api/destinations')) return new Response('[]', { status: 200 });
				if (init?.method === 'PUT') return new Response(JSON.stringify({ ID: 'ride-1' }), { status: 200 });
				return new Response(
					JSON.stringify({
						ID: 'ride-1',
						Origin: 'Saillans',
						Destination: 'Crest',
						DepartureAt: '2030-06-01T09:00:00Z',
						Flexibility: 30
					}),
					{ status: 200 }
				);
			})
		);

		const { container } = render(Edit);
		await vi.waitFor(() => expect(container.querySelector('#edit-ride-form')).toBeTruthy());
		await fireEvent.submit(container.querySelector('#edit-ride-form')!);
		await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/my-rides'));

		const put = calls.find((c) => c.method === 'PUT');
		expect(put?.url).toContain('/api/rides/ride-1');
		expect(JSON.parse(put!.body!)).toMatchObject({
			phone: '0612345678',
			origin: 'Saillans',
			destination: 'Crest',
			flexibility: 30
		});
	});
});
