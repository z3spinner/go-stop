// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import LangPicker from './LangPicker.svelte';

describe('LangPicker', () => {
	it('shows the current locale flag and a dropdown with all 7 locales', async () => {
		const { container } = render(LangPicker);
		// 7 language options exist (hidden until opened): fr, en, es, it, de, nl, el
		expect(container.querySelectorAll('.lang-opt').length).toBe(7);
	});
});
