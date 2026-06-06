// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import RideCard from './RideCard.svelte';

const ride = {
	ID: '42',
	DriverName: 'Alice',
	Origin: 'Saillans',
	Destination: 'Crest',
	DepartureAt: '2030-06-15T08:00:00Z',
	Flexibility: 0,
	InterestCount: 0
} as never;

beforeEach(() => localStorage.clear());

describe('RideCard', () => {
	it('links the card body to the ride detail page', () => {
		const { container } = render(RideCard, { props: { ride } });
		const link = container.querySelector('a.card-detail-link') as HTMLAnchorElement;
		expect(link).toBeTruthy();
		expect(link.getAttribute('href')).toBe('/rides/42');
	});

	it('marks an own ride with a badge and a manage link, and hides the contact button', () => {
		const { container } = render(RideCard, { props: { ride, isOwn: true } });
		const badge = container.querySelector('.tag-your-ride') as HTMLElement;
		expect(badge).toBeTruthy();
		expect(badge.textContent).toMatch(/Your ride|Votre trajet/);
		const manage = container.querySelector('a.ride-manage-link') as HTMLAnchorElement;
		expect(manage).toBeTruthy();
		expect(manage.getAttribute('href')).toBe('/my-rides#card-42');
		expect(container.querySelector('.btn-interest')).toBeNull();
	});

	it('renders the contact action and no own-ride badge by default', () => {
		const { container } = render(RideCard, { props: { ride } });
		expect(container.querySelector('.tag-your-ride')).toBeNull();
		expect(container.querySelector('.ride-manage-link')).toBeNull();
		expect(container.querySelector('.btn-interest')).toBeTruthy();
	});
});
