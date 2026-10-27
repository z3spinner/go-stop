// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import BellButton from './BellButton.svelte';
import { userPhone } from '#lib/stores.js';
import { inboxUnread } from '#lib/inbox.js';
import { pushState } from '#lib/pwa.js';
import { notifModalState } from '#lib/notifModal.js';

const goto = vi.fn();
vi.mock('$app/navigation', () => ({ goto: (...a: unknown[]) => goto(...a) }));

beforeEach(() => {
	localStorage.clear();
	goto.mockReset();
	userPhone.set('');
	inboxUnread.set(0);
	pushState.set('default');
	notifModalState.set(null);
});

describe('BellButton', () => {
	it('without a phone, opens push setup (no inbox yet)', async () => {
		const { container } = render(BellButton);
		await fireEvent.click(container.querySelector('.btn-bell')!);
		expect(goto).not.toHaveBeenCalled();
		expect(get(notifModalState)).toBe('default');
	});

	it('with a phone, opens the inbox', async () => {
		userPhone.set('0611111111');
		const { container } = render(BellButton);
		await fireEvent.click(container.querySelector('.btn-bell')!);
		expect(goto).toHaveBeenCalledWith('/notifications');
	});

	it('shows the unread badge, capped at 9+', async () => {
		userPhone.set('0611111111');
		inboxUnread.set(3);
		const { container } = render(BellButton);
		expect(container.querySelector('.bell-badge')?.textContent).toBe('3');
		inboxUnread.set(42);
		await tick();
		expect(container.querySelector('.bell-badge')?.textContent).toBe('9+');
	});

	it('hides the badge when nothing is unread', () => {
		userPhone.set('0611111111');
		const { container } = render(BellButton);
		expect(container.querySelector('.bell-badge')).toBeNull();
	});

	it('on iOS Safari with a phone, shows the bell and the add-to-home hint', () => {
		userPhone.set('0611111111');
		pushState.set('ios-browser');
		const { container } = render(BellButton);
		expect(container.querySelector('.btn-bell')).not.toBeNull();
		expect(container.querySelector('.bell-activate-label')).not.toBeNull();
	});
});
