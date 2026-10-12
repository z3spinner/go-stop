// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import NumberStepper from './NumberStepper.svelte';

describe('NumberStepper', () => {
	it('shows the value and increments/decrements within bounds', async () => {
		const { container, getByLabelText } = render(NumberStepper, { props: { value: 4, min: 1, max: 14 } });
		const input = container.querySelector('input') as HTMLInputElement;
		expect(input.value).toBe('4');
		await fireEvent.click(getByLabelText('increase'));
		expect(input.value).toBe('5');
		await fireEvent.click(getByLabelText('decrease'));
		expect(input.value).toBe('4');
	});

	it('disables decrease at min and increase at max', () => {
		const atMin = render(NumberStepper, { props: { value: 1, min: 1, max: 14 } });
		expect((atMin.getByLabelText('decrease') as HTMLButtonElement).disabled).toBe(true);
		atMin.unmount();
		const atMax = render(NumberStepper, { props: { value: 14, min: 1, max: 14 } });
		expect((atMax.getByLabelText('increase') as HTMLButtonElement).disabled).toBe(true);
	});

	it('clamps a typed out-of-range value on change', async () => {
		const { container } = render(NumberStepper, { props: { value: 4, min: 1, max: 14 } });
		const input = container.querySelector('input') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: '99' } });
		await fireEvent.change(input);
		expect(input.value).toBe('14');
	});

	it('re-syncs the input when a typed value clamps to the current value', async () => {
		const { container } = render(NumberStepper, { props: { value: 14, min: 1, max: 14 } });
		const input = container.querySelector('input') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: '99' } });
		await fireEvent.change(input);
		expect(input.value).toBe('14');
	});
});
