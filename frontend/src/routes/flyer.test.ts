// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import { config } from '$lib/config';

// qrcode is async and DOM-free; stub it so the component renders deterministically.
vi.mock('qrcode', () => ({
	default: { toString: vi.fn().mockResolvedValue('<svg data-testid="qr-svg"></svg>') }
}));

import Flyer from './flyer/+page.svelte';

beforeEach(() => {
	config.set({ siteName: 'Go-Stop Saillans', returnDelayHours: 2 });
});

describe('flyer', () => {
	it('shows the configured site name', () => {
		const { getByText } = render(Flyer);
		expect(getByText('Go-Stop Saillans')).toBeTruthy();
	});

	it('shows the host derived from the page origin', async () => {
		const { getByText } = render(Flyer);
		await waitFor(() =>
			expect(getByText((t) => t.includes(window.location.host))).toBeTruthy()
		);
	});

	it('switches flyer language on demand', async () => {
		const { getByText } = render(Flyer);
		await fireEvent.click(getByText('EN'));
		expect(getByText('Need a ride?')).toBeTruthy();
		await fireEvent.click(getByText('FR'));
		expect(getByText((t) => t.includes("Besoin d'un stop"))).toBeTruthy();
	});

	it('print button calls window.print', async () => {
		const printSpy = vi.fn();
		window.print = printSpy;
		const { getByText } = render(Flyer);
		await fireEvent.click(getByText('FR')); // make the print label deterministic ("Imprimer")
		await fireEvent.click(getByText('Imprimer'));
		expect(printSpy).toHaveBeenCalled();
	});

	it('derives the "Made in" place from the site name', async () => {
		// beforeEach sets siteName "Go-Stop Saillans" → place "Saillans".
		const { getByText } = render(Flyer);
		await fireEvent.click(getByText('EN'));
		expect(getByText((t) => t.includes('Made in Saillans'))).toBeTruthy();
	});

	it('hides the accolade when the site name has no place', () => {
		config.set({ siteName: 'Go-Stop', returnDelayHours: 2 });
		const { queryByText } = render(Flyer);
		expect(queryByText((t) => /made in|fait à/i.test(t))).toBeNull();
	});
});
