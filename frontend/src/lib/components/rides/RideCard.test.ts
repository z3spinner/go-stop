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
});
