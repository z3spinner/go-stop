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
	it('posts one ride with the typed origin/destination when not repeating', async () => {
		const { container } = render(RideForm);
		await fillBase(container);
		await fireEvent.submit(container.querySelector('#ride-form')!);
		await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
		const body = post.mock.calls[0][0] as { origin: string; destination: string };
		expect(body.origin).toBe('Saillans');
		expect(body.destination).toBe('Crest');
	});
});
