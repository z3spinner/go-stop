// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { goto } from '$app/navigation';
import { userName, userPhone } from '$lib/stores';
import RequestFeedCard from './RequestFeedCard.svelte';
import type { PublicRequest } from '$lib/types';

const offerContact = vi.hoisted(() => vi.fn(async () => null));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$lib/api', () => ({ api: { requests: { offerContact } } }));
vi.mock('$lib/profileModal', () => ({ openProfileModal: vi.fn() }));
vi.mock('$lib/paraglide/messages', () => ({
	m: {
		alertAnytimeLabel: () => 'Anytime',
		at: () => 'at',
		btnDriveThis: () => 'I can drive this',
		btnShareContact: () => 'Share my contact',
		contactOfferSent: () => 'Contact shared ✓',
		flexLabel30: () => '±30 min',
		flexLabel60: () => '±60 min',
		flexLabelExact: () => 'Exact'
	}
}));
const gotoMock = vi.mocked(goto);

const ZERO = '0001-01-01T00:00:00Z';
const base: PublicRequest = {
	ID: 'rq1', SearcherName: 'Bob', Origin: 'Saillans', Destination: 'Crest',
	Date: ZERO, DepartureAt: ZERO, Flexibility: 0
};

beforeEach(() => {
	gotoMock.mockClear();
	offerContact.mockClear();
	localStorage.clear();
	userName.set('Alice');
	userPhone.set('06 11 00 00 01');
});

describe('RequestFeedCard', () => {
	it('renders an anytime request with the anytime label', () => {
		const { container } = render(RequestFeedCard, { props: { request: { ...base, Date: ZERO, DepartureAt: ZERO } } });
		expect(container.querySelector('.tag-anytime')).toBeTruthy();
		expect(container.textContent).toContain('Bob');
	});

	it('renders a date-only "day" request as a date, not anytime', () => {
		const { container } = render(RequestFeedCard, {
			props: { request: { ...base, Date: '2030-06-15T00:00:00Z', DepartureAt: ZERO } }
		});
		expect(container.querySelector('.tag-anytime')).toBeNull();
	});

	it('renders a time request with the formatted time, not anytime', () => {
		const { container } = render(RequestFeedCard, {
			props: { request: { ...base, Date: '2030-06-15T00:00:00Z', DepartureAt: '2030-06-15T08:30:00Z', Flexibility: 30 } }
		});
		expect(container.querySelector('.tag-anytime')).toBeNull();
		expect(container.textContent).toMatch(/\d{1,2}:\d{2}/); // a concrete time (TZ-independent)
	});

	it('"I can drive this" navigates to a prefilled post-ride URL', async () => {
		const { container } = render(RequestFeedCard, { props: { request: { ...base, Date: ZERO, DepartureAt: ZERO } } });
		await fireEvent.click(container.querySelector('.btn-drive-this')!);
		expect(gotoMock).toHaveBeenCalledTimes(1);
		const url = gotoMock.mock.calls[0][0] as string;
		expect(url).toContain('/post-ride?');
		expect(url).toContain('origin=Saillans');
		expect(url).toContain('destination=Crest');
		expect(url).not.toContain('departure_at'); // anytime → no concrete instant
	});

	it('a one-off time request seeds departure_at in the CTA URL', async () => {
		const { container } = render(RequestFeedCard, {
			props: { request: { ...base, Date: '2030-06-15T00:00:00Z', DepartureAt: '2030-06-15T08:30:00Z', Flexibility: 30 } }
		});
		await fireEvent.click(container.querySelector('.btn-drive-this')!);
		expect(gotoMock.mock.calls[0][0] as string).toContain('departure_at=');
	});

	it('marks a request as already shared after offering contact', async () => {
		const { container } = render(RequestFeedCard, { props: { request: { ...base } } });
		const shareButton = container.querySelector('.btn-share-contact') as HTMLButtonElement;
		await fireEvent.click(shareButton);
		expect(offerContact).toHaveBeenCalledWith('rq1', '0611000001', 'Alice');
		expect(shareButton).toBeDisabled();
		expect(shareButton.textContent).toContain('Contact shared ✓');
		expect(localStorage.getItem('contact_offer_0611000001_rq1')).toBe('1');
	});

	it('restores the shared state from localStorage for the current phone', () => {
		localStorage.setItem('contact_offer_0611000001_rq1', '1');
		const { container } = render(RequestFeedCard, { props: { request: { ...base } } });
		const shareButton = container.querySelector('.btn-share-contact') as HTMLButtonElement;
		expect(shareButton).toBeDisabled();
		expect(shareButton.textContent).toContain('Contact shared ✓');
	});
});
