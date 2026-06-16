// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

const post = vi.fn(async (_body: unknown) => ({}));
vi.mock('$lib/api', () => ({ api: { rides: { post: (b: unknown) => post(b) } } }));

import RideForm from './RideForm.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => {
	post.mockClear();
	userName.set('Marie');
	userPhone.set('0612345678');
});

async function fillBase(container: HTMLElement) {
	await fireEvent.input(container.querySelector('input[name=origin]')!, { target: { value: 'Saillans' } });
	await fireEvent.input(container.querySelector('input[name=destination]')!, { target: { value: 'Crest' } });
	await fireEvent.input(container.querySelector('input[name=departure_at]')!, { target: { value: '2030-06-03T08:00' } });
}

describe('RideForm repeat', () => {
	it('posts one ride when not repeating (regression)', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.submit(container.querySelector('#ride-form')!);
		await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
	});

	it('daily × 3 posts three rides on consecutive days at the same time', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.change(container.querySelector('#repeat-frequency')!, { target: { value: 'daily' } });
		await fireEvent.input(container.querySelector('#repeat-count')!, { target: { value: '3' } });
		await fireEvent.submit(container.querySelector('#ride-form')!);

		await waitFor(() => expect(post).toHaveBeenCalledTimes(3));
		const days = post.mock.calls.map((c) => new Date((c[0] as { departure_at: string }).departure_at).getDate());
		expect(days).toEqual([3, 4, 5]);
		const hours = post.mock.calls.map((c) => new Date((c[0] as { departure_at: string }).departure_at).getHours());
		expect(hours).toEqual([8, 8, 8]);
	});

	it('repeat with return posts both legs per occurrence', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.click(container.querySelector('#btn-return')!);
		await fireEvent.change(container.querySelector('#repeat-frequency')!, { target: { value: 'daily' } });
		await fireEvent.input(container.querySelector('#repeat-count')!, { target: { value: '2' } });
		await fireEvent.submit(container.querySelector('#ride-form')!);

		// 2 outbound + 2 return
		await waitFor(() => expect(post).toHaveBeenCalledTimes(4));
		const returns = post.mock.calls.filter((c) => (c[0] as { origin: string }).origin === 'Crest');
		expect(returns).toHaveLength(2);
	});
});
