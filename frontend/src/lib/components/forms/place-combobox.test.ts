// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import PlaceCombobox from './PlaceCombobox.svelte';

describe('PlaceCombobox', () => {
	it('renders an input carrying name and required', () => {
		const { container } = render(PlaceCombobox, {
			props: { value: '', items: ['Saillans', 'Crest'], name: 'origin', required: true }
		});
		const input = container.querySelector('input[name=origin]') as HTMLInputElement;
		expect(input).toBeTruthy();
		expect(input.required).toBe(true);
	});

	it('reflects a free-text value not present in items', async () => {
		const { container } = render(PlaceCombobox, {
			props: { value: '', items: ['Saillans', 'Crest'], name: 'origin' }
		});
		const input = container.querySelector('input[name=origin]') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: 'Nowheresville' } });
		expect(input.value).toBe('Nowheresville');
	});

	it('shows a passed value', () => {
		const { container } = render(PlaceCombobox, {
			props: { value: 'Saillans', items: ['Saillans', 'Crest'], name: 'origin' }
		});
		const input = container.querySelector('input[name=origin]') as HTMLInputElement;
		expect(input.value).toBe('Saillans');
	});
});
