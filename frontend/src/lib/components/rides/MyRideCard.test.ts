// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';

const { listInterests } = vi.hoisted(() => ({ listInterests: vi.fn() }));
vi.mock('$lib/api', () => ({
	api: {
		rides: {
			listMatchingRequests: vi.fn(async () => []),
			listInterests,
		},
	},
}));

import MyRideCard from './MyRideCard.svelte';

const ride = {
	ID: 'r1', Origin: 'Saillans', Destination: 'Crest',
	DepartureAt: '2030-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;

beforeEach(() => listInterests.mockReset());

describe('MyRideCard name display', () => {
	it('shows "{name} wants a ride" for a pending interest', async () => {
		listInterests.mockResolvedValue([{ id: 'i1', status: 'pending', searcher_name: 'Marie' }]);
		render(MyRideCard, { props: { ride, phone: '5550001' } });
		expect(await screen.findByText(/Marie/)).toBeInTheDocument();
		expect(screen.getByText(/wants a ride|demande un trajet/)).toBeInTheDocument();
	});

	it('shows name and phone for an accepted interest', async () => {
		listInterests.mockResolvedValue([
			{ id: 'i2', status: 'accepted', searcher_name: 'Marie', searcher_phone: '0612345678' },
		]);
		const { container } = render(MyRideCard, { props: { ride, phone: '5550001' } });
		await screen.findByText(/Marie/);
		expect(container.querySelector('a[href="tel:0612345678"]')).toBeInTheDocument();
		expect(container.querySelector('.interest-accepted')!.textContent).toMatch(/Marie\s*—\s*0612345678/);
	});

	it('falls back to a placeholder when a pending interest has no name', async () => {
		listInterests.mockResolvedValue([{ id: 'i3', status: 'pending', searcher_name: '' }]);
		render(MyRideCard, { props: { ride, phone: '5550001' } });
		expect(await screen.findByText(/Someone|Quelqu'un/)).toBeInTheDocument();
	});
});
