// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import FlexibilitySelect from './FlexibilitySelect.svelte';
import { m } from '$lib/paraglide/messages';

describe('FlexibilitySelect', () => {
	it('renders the label for the current value on the trigger', () => {
		const { getByText } = render(FlexibilitySelect, { props: { value: 30 } });
		expect(getByText(m.flex30())).toBeTruthy();
	});

	it('renders the exact label for value 0', () => {
		const { getByText } = render(FlexibilitySelect, { props: { value: 0 } });
		expect(getByText(m.flexExact())).toBeTruthy();
	});
});
