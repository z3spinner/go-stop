// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ContactOrInterest from './ContactOrInterest.svelte';
import { get } from 'svelte/store';
import { userName, userPhone } from '$lib/stores';
import { profileModalState } from '$lib/profileModal';

beforeEach(() => { localStorage.clear(); userName.set('Bob'); userPhone.set('0622000002'); profileModalState.set(null); });
afterEach(() => vi.unstubAllGlobals());

const ride = { ID: '42', DriverName: 'Alice', Origin: 'Saillans', Destination: 'Crest', Flexibility: 0 } as any;

describe('ContactOrInterest', () => {
	it('renders a request-contact button with data-ride-id', () => {
		const { container } = render(ContactOrInterest, { props: { ride } });
		const btn = container.querySelector('.btn-interest') as HTMLElement;
		expect(btn).toBeTruthy();
		expect(btn.dataset.rideId).toBe('42');
	});

	it('expressing interest POSTs and stores interest_<id> in localStorage', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ id: 'int1', status: 'pending' }), { status: 201 })));
		const { container } = render(ContactOrInterest, { props: { ride } });
		await fireEvent.click(container.querySelector('.btn-interest')!);
		await vi.waitFor(() => expect(localStorage.getItem('interest_42')).toBe('int1'));
		expect(container.querySelector('.interest-state')!.textContent).toMatch(/envoyée|sent/i);
	});

	it('shows a tel link when a contact phone is already known', () => {
		const { container } = render(ContactOrInterest, { props: { ride, contactPhone: '0611000001' } });
		expect(container.querySelector('a[href="tel:0611000001"]')).toBeTruthy();
	});

	it('cancelling a pending request DELETEs it and clears interest_<id>', async () => {
		localStorage.setItem('interest_42', 'int1'); // start in the pending state
		const fetchMock = vi.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		vi.stubGlobal('fetch', fetchMock);
		const { container } = render(ContactOrInterest, { props: { ride } });

		const cancelBtn = container.querySelector('.btn-interest-cancel') as HTMLElement;
		expect(cancelBtn).toBeTruthy();
		await fireEvent.click(cancelBtn);

		await vi.waitFor(() => expect(localStorage.getItem('interest_42')).toBeNull());
		// reverted to the plain request-contact button; the resend/cancel are gone
		expect(container.querySelector('.btn-interest-resend')).toBeNull();
		expect(container.querySelector('.btn-interest-cancel')).toBeNull();
		expect(container.querySelector('.interest-state')!.textContent).toMatch(/annulée|cancelled/i);

		const [url, init] = fetchMock.mock.calls[0];
		expect(String(url)).toContain('/api/interests/int1');
		expect(init?.method).toBe('DELETE');
	});

	it('opens the profile modal instead of sending when the profile is incomplete', async () => {
		userName.set(''); // incomplete: no name
		userPhone.set('0622000002');
		const fetchMock = vi.fn(async () => new Response(JSON.stringify({ id: 'int1', status: 'pending' }), { status: 201 }));
		vi.stubGlobal('fetch', fetchMock);

		const { container } = render(ContactOrInterest, { props: { ride } });
		await fireEvent.click(container.querySelector('.btn-interest')!);

		expect(get(profileModalState)).toBeTypeOf('function'); // modal opened
		expect(fetchMock).not.toHaveBeenCalled(); // nothing sent yet
	});

	it('resend also opens the profile modal when the profile is incomplete', async () => {
		localStorage.setItem('interest_42', 'int1'); // start in pending state
		userName.set('');
		userPhone.set('0622000002');
		const fetchMock = vi.fn(async () => new Response(JSON.stringify({ id: 'int1', status: 'pending' }), { status: 201 }));
		vi.stubGlobal('fetch', fetchMock);

		const { container } = render(ContactOrInterest, { props: { ride } });
		const resend = container.querySelector('.btn-interest-resend') as HTMLElement;
		expect(resend).toBeTruthy();
		await fireEvent.click(resend);

		expect(get(profileModalState)).toBeTypeOf('function');
		expect(fetchMock).not.toHaveBeenCalled();
	});
});
