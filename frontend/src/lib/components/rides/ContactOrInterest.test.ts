import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ContactOrInterest from './ContactOrInterest.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => { localStorage.clear(); userName.set('Bob'); userPhone.set('0622000002'); });
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
});
