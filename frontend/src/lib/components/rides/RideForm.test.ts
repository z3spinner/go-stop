// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import RideForm from './RideForm.svelte';
import { config } from '$lib/config';

beforeEach(() => { localStorage.clear(); config.set({ siteName: 'Go-Stop', returnDelayHours: 2 }); });

describe('RideForm', () => {
	it('return toggle defaults return time to outbound + returnDelayHours', async () => {
		const { container } = render(RideForm, { props: { destinations: [] } });
		const dep = container.querySelector('input[name=departure_at]') as HTMLInputElement;
		await fireEvent.input(dep, { target: { value: '2030-12-01T09:00' } });
		await fireEvent.click(container.querySelector('#btn-return')!);
		const ret = container.querySelector('input[name=return_departure_at]') as HTMLInputElement;
		expect(ret.value).toBe('2030-12-01T11:00');
	});

	it('prefills origin/destination/departure from props ("I can drive this")', () => {
		const { container } = render(RideForm, {
			props: { destinations: [], origin: 'Saillans', destination: 'Crest', departureAt: '2030-06-15T08:30' }
		});
		expect((container.querySelector('input[name=origin]') as HTMLInputElement).value).toBe('Saillans');
		expect((container.querySelector('input[name=destination]') as HTMLInputElement).value).toBe('Crest');
		expect((container.querySelector('input[name=departure_at]') as HTMLInputElement).value).toBe('2030-06-15T08:30');
	});
});
