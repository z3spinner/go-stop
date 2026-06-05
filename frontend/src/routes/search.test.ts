// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import Search from './search/+page.svelte';

afterEach(() => vi.unstubAllGlobals());

vi.mock('$app/state', () => ({
	page: { url: new URL('http://localhost/search?origin=Saillans&destination=Crest') }
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

describe('search', () => {
	it('renders two result columns after an auto-submitted query', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('[]', { status: 200 })));
		const { container } = render(Search);
		await vi.waitFor(() => expect(container.querySelectorAll('.results-col-header').length).toBe(2));
	});
});
