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

describe('MyRideCard delete gated on feedback', () => {
	it('disables delete on a past, unanswered ride and shows the question', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		expect(container.querySelector('.feedback-prompt')).toBeInTheDocument();
		expect(container.querySelector('.btn-delete')).toBeDisabled();
	});

	it('selecting an answer enables delete and shows the selection, without recording yet', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-fb-yes')!);
		// nothing committed to the server on selection
		expect(feedback).not.toHaveBeenCalled();
		// the choice stays visible and marked selected (no "Merci" collapse)
		expect(container.querySelector('.feedback-prompt')).toBeInTheDocument();
		expect(container.querySelector('.btn-fb-yes')).toHaveClass('selected');
		expect(container.querySelector('.btn-fb-no')).not.toHaveClass('selected');
		// delete is now enabled
		expect(container.querySelector('.btn-delete')).not.toBeDisabled();
	});

	it('lets the user change the choice and commits the final one on delete', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-fb-yes')!);
		await fireEvent.click(container.querySelector('.btn-fb-no')!); // change of mind
		expect(container.querySelector('.btn-fb-no')).toHaveClass('selected');
		expect(container.querySelector('.btn-fb-yes')).not.toHaveClass('selected');
		expect(feedback).not.toHaveBeenCalled(); // still not recorded
		await fireEvent.click(container.querySelector('.btn-delete')!);
		// committed once, with the final choice
		expect(feedback).toHaveBeenCalledTimes(1);
		expect(feedback).toHaveBeenCalledWith('rp', 'p', false);
		expect(del).toHaveBeenCalledWith('rp', 'p');
	});

	it('deletes a future ride immediately (no question, no feedback)', async () => {
		const { container } = render(MyRideCard, { props: { ride: futureRide, phone: 'p' } });
		expect(container.querySelector('.feedback-prompt')).toBeNull();
		expect(container.querySelector('.btn-delete')).not.toBeDisabled();
		await fireEvent.click(container.querySelector('.btn-delete')!);
		expect(feedback).not.toHaveBeenCalled();
		expect(del).toHaveBeenCalledWith('r1', 'p');
	});
});
