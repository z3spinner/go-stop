// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

const { listInterests, del, feedback } = vi.hoisted(() => ({
	listInterests: vi.fn(),
	del: vi.fn(async () => null),
	feedback: vi.fn(async () => null),
}));
vi.mock('$lib/api', () => ({
	api: {
		rides: {
			listMatchingRequests: vi.fn(async () => []),
			listInterests,
			del,
			feedback,
		},
	},
}));

import MyRideCard from './MyRideCard.svelte';

const futureRide = {
	ID: 'r1', Origin: 'Saillans', Destination: 'Crest',
	DepartureAt: '2030-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;
const pastRide = {
	ID: 'rp', Origin: 'A', Destination: 'B',
	DepartureAt: '2020-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;

beforeEach(() => {
	listInterests.mockReset();
	listInterests.mockResolvedValue([]);
	del.mockClear();
	feedback.mockClear();
});

describe('MyRideCard name display', () => {
	it('shows "{name} wants a ride" for a pending interest', async () => {
		listInterests.mockResolvedValue([{ id: 'i1', status: 'pending', searcher_name: 'Marie' }]);
		render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		expect(await screen.findByText(/Marie/)).toBeInTheDocument();
		expect(screen.getByText(/wants a ride|demande un trajet/)).toBeInTheDocument();
	});

	it('shows name and phone for an accepted interest', async () => {
		listInterests.mockResolvedValue([
			{ id: 'i2', status: 'accepted', searcher_name: 'Marie', searcher_phone: '0612345678' },
		]);
		const { container } = render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		await screen.findByText(/Marie/);
		expect(container.querySelector('a[href="tel:0612345678"]')).toBeInTheDocument();
		expect(container.querySelector('.interest-accepted')!.textContent).toMatch(/Marie\s*—\s*0612345678/);
	});

	it('falls back to a placeholder when a pending interest has no name', async () => {
		listInterests.mockResolvedValue([{ id: 'i3', status: 'pending', searcher_name: '' }]);
		render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		expect(await screen.findByText(/Someone|Quelqu'un/)).toBeInTheDocument();
	});
});

describe('MyRideCard ask-on-delete', () => {
	it('asks the came-along question when deleting a past, unanswered ride', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		// the confirm block appears; nothing deleted yet
		expect(container.querySelector('.delete-confirm')).toBeInTheDocument();
		expect(del).not.toHaveBeenCalled();
		// answering yes records feedback then deletes
		await fireEvent.click(container.querySelector('.btn-del-yes')!);
		expect(feedback).toHaveBeenCalledWith('rp', 'p', true);
		expect(del).toHaveBeenCalledWith('rp', 'p');
	});

	it('records "no" then deletes', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		await fireEvent.click(container.querySelector('.btn-del-no')!);
		expect(feedback).toHaveBeenCalledWith('rp', 'p', false);
		expect(del).toHaveBeenCalledWith('rp', 'p');
	});

	it('deletes a future ride silently (no question)', async () => {
		const { container } = render(MyRideCard, { props: { ride: futureRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		expect(container.querySelector('.delete-confirm')).toBeNull();
		expect(feedback).not.toHaveBeenCalled();
		expect(del).toHaveBeenCalledWith('r1', 'p');
	});
});
