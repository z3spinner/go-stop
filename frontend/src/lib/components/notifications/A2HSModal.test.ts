// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import A2HSModal from './A2HSModal.svelte';
import { a2hsModalOpen } from '$lib/a2hs';
import { m } from '$lib/paraglide/messages';

const SAFARI = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1';
const INSTAGRAM = 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 400.0.0.25.86';

/** Pretend to be this browser (and, optionally, give it a clipboard). */
function asBrowser(ua: string, clipboard?: { writeText: (s: string) => Promise<void> }) {
	Object.defineProperty(navigator, 'userAgent', { value: ua, configurable: true });
	if (clipboard) Object.defineProperty(navigator, 'clipboard', { value: clipboard, configurable: true });
}

afterEach(() => {
	a2hsModalOpen.set(false);
	// Drop the own-property overrides so jsdom's own getters show through again.
	delete (navigator as unknown as Record<string, unknown>).userAgent;
	delete (navigator as unknown as Record<string, unknown>).clipboard;
	vi.restoreAllMocks();
});

describe('A2HSModal', () => {
	it('in Safari, walks through adding the page to the Home Screen', async () => {
		asBrowser(SAFARI);
		render(A2HSModal);
		a2hsModalOpen.set(true);
		expect(await screen.findByText(m.a2hsStep1())).toBeInTheDocument();
		expect(screen.getByText(m.a2hsStep2())).toBeInTheDocument();
		expect(screen.queryByText(m.a2hsInAppBody())).toBeNull();
	});

	it("inside another app's browser, says to open the page in Safari and copies the link", async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		asBrowser(INSTAGRAM, { writeText });
		render(A2HSModal);
		a2hsModalOpen.set(true);
		expect(await screen.findByText(m.a2hsInAppBody())).toBeInTheDocument();
		expect(screen.queryByText(m.a2hsStep1())).toBeNull();
		expect(screen.getByText(location.origin)).toBeInTheDocument();

		await fireEvent.click(screen.getByRole('button', { name: m.a2hsCopyLink() }));
		expect(writeText).toHaveBeenCalledWith(location.origin);
		await waitFor(() => expect(screen.getByText(m.a2hsCopied())).toBeInTheDocument());
	});
});
