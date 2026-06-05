// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SeekerRow from './SeekerRow.svelte';

afterEach(() => vi.unstubAllGlobals());
const req = { ID: 'rq1', SearcherName: 'Bob', Origin: 'Saillans', Destination: 'Crest', DepartureAt: '2030-06-15T08:00:00Z', Flexibility: 0 } as any;

describe('SeekerRow', () => {
	it('shows the searcher name, carries data attrs, and pings on click', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response(null, { status: 204 })));
		const { container } = render(SeekerRow, { props: { request: req, rideId: 'r9', driverPhone: '0611000001' } });
		const btn = container.querySelector('.btn-ping-searcher') as HTMLButtonElement;
		expect(container.querySelector('.seeker-row')!.textContent).toContain('Bob');
		expect(btn.dataset.reqId).toBe('rq1');
		expect(btn.dataset.rideId).toBe('r9');
		await fireEvent.click(btn);
		await vi.waitFor(() => expect(btn.disabled).toBe(true));
	});
});
