// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import Home from './+page.svelte';

afterEach(() => vi.unstubAllGlobals());

describe('home', () => {
	it('renders hero CTAs and ghost nav', () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Home);
		expect(container.querySelector('button.btn-primary')?.textContent).toMatch(/Je conduis|driving/);
		expect(container.querySelector('button.btn-secondary')).toBeTruthy();
		// Me · My rides · My searches (share lives in the header/TopBar, not here)
		expect(container.querySelectorAll('.btn-ghost-inline').length).toBe(3);
	});

	it('renders the Available / Requested swipe tabs and both panels', () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Home);
		expect(container.querySelectorAll('[data-slot="tabs-trigger"]').length).toBe(2);
		expect(container.querySelectorAll('.feed-panel').length).toBe(2);
	});
});
