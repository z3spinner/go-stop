// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import PullToRefresh from './PullToRefresh.svelte';

// Toggle standalone detection per test.
const { standalone } = vi.hoisted(() => ({ standalone: { value: true } }));
vi.mock('$lib/pwa', () => ({ isStandalone: () => standalone.value }));

function emit(type: string, y: number, x = 0) {
	const e = new Event(type, { bubbles: true, cancelable: true });
	Object.defineProperty(e, 'touches', { value: [{ clientX: x, clientY: y }] });
	window.dispatchEvent(e);
}

// pull = (dy) * 0.5; threshold is 70 → need dy >= 140 to trigger.
async function pullBy(dy: number, dx = 0) {
	emit('touchstart', 100, 0);
	emit('touchmove', 100 + dy, dx);
	emit('touchend', 100 + dy, dx);
	await Promise.resolve();
}

beforeEach(() => {
	standalone.value = true;
	window.scrollY = 0;
});

describe('PullToRefresh', () => {
	it('triggers refresh when pulled past the threshold', async () => {
		const onrefresh = vi.fn(() => Promise.resolve());
		render(PullToRefresh, { onrefresh });

		await pullBy(160); // 160 * 0.5 = 80 ≥ 70

		expect(onrefresh).toHaveBeenCalledTimes(1);
	});

	it('does not trigger when the pull is below the threshold', async () => {
		const onrefresh = vi.fn(() => Promise.resolve());
		render(PullToRefresh, { onrefresh });

		await pullBy(80); // 80 * 0.5 = 40 < 70

		expect(onrefresh).not.toHaveBeenCalled();
	});

	it('ignores horizontal swipes (lets carousels scroll)', async () => {
		const onrefresh = vi.fn(() => Promise.resolve());
		render(PullToRefresh, { onrefresh });

		// Mostly-horizontal move past the vertical threshold distance.
		await pullBy(160, 300);

		expect(onrefresh).not.toHaveBeenCalled();
	});

	it('does nothing in a normal browser tab (not standalone)', async () => {
		standalone.value = false;
		const onrefresh = vi.fn(() => Promise.resolve());
		render(PullToRefresh, { onrefresh });

		await pullBy(160);

		expect(onrefresh).not.toHaveBeenCalled();
	});

	it('does not arm when the page is scrolled down', async () => {
		window.scrollY = 200;
		const onrefresh = vi.fn(() => Promise.resolve());
		render(PullToRefresh, { onrefresh });

		await pullBy(160);

		expect(onrefresh).not.toHaveBeenCalled();
	});
});
