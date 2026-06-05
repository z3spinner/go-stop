// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AlertForm from './AlertForm.svelte';

beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());

describe('AlertForm', () => {
	it('renders #notify-form with 4 mode buttons', () => {
		const { container } = render(AlertForm, { props: { origin: 'Saillans', destination: 'Crest' } });
		expect(container.querySelector('#notify-form')).toBeTruthy();
		expect(container.querySelectorAll('.btn-mode').length).toBe(4);
	});

	it('mode=time with date+time posts a departure_at instant', async () => {
		let body: any;
		vi.stubGlobal('fetch', vi.fn(async (_u: string, init: RequestInit) => { body = JSON.parse(init.body as string); return new Response(JSON.stringify({ ID: 'x' }), { status: 201 }); }));
		const { container } = render(AlertForm, { props: { origin: 'Saillans', destination: 'Crest' } });
		await fireEvent.input(container.querySelector('input[name=searcher_name]')!, { target: { value: 'Bob' } });
		await fireEvent.input(container.querySelector('input[name=phone]')!, { target: { value: '0622000002' } });
		await fireEvent.input(container.querySelector('input[name=alert_date]')!, { target: { value: '2030-06-15' } });
		await fireEvent.input(container.querySelector('input[name=alert_time]')!, { target: { value: '09:30' } });
		await fireEvent.submit(container.querySelector('#notify-form')!);
		await vi.waitFor(() => expect(body).toBeTruthy());
		expect(body.departure_at).toBeTruthy();
		expect(body.searcher_name).toBe('Bob');
	});
});
