// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import About from './about/+page.svelte';

describe('about', () => {
	it('links to the printable flyer', () => {
		const { container } = render(About);
		const link = container.querySelector('a[href="/flyer"]');
		expect(link).toBeTruthy();
	});
});
