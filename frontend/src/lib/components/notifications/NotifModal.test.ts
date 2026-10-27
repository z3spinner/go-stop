// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import NotifModal from './NotifModal.svelte';
import { notifModalState } from '#lib/notifModal.js';
import { api } from '#lib/api.js';
import { config } from '#lib/config.js';
import { userPhone } from '#lib/stores.js';
import { m } from '#lib/paraglide/messages.js';

beforeEach(() => notifModalState.set(null));
afterEach(() => {
	vi.restoreAllMocks();
	config.set({ siteName: 'Go-Stop', returnDelayHours: 2 });
	userPhone.set('');
});

describe('NotifModal', () => {
	it('is closed when state is null', () => {
		render(NotifModal);
		// shadcn Dialog renders nothing (no role=dialog) while closed.
		expect(screen.queryByRole('dialog')).toBeNull();
	});
	it('default state shows enable + skip buttons', async () => {
		render(NotifModal);
		notifModalState.set('default');
		expect(await screen.findByText(/Activer les notifications|Enable notifications/)).toBeInTheDocument();
	});
	it('subscribed state shows test + re-configure buttons', async () => {
		render(NotifModal);
		notifModalState.set('subscribed');
		expect(await screen.findByText(/Envoyer un test|Send a test/)).toBeInTheDocument();
		expect(screen.getByText(/Reconfigurer|Re-configure/)).toBeInTheDocument();
	});
});

describe('NotifModal options: morning digest', () => {
	beforeEach(() => {
		config.set({ siteName: 'Go-Stop', returnDelayHours: 2, morningDigestHour: 8 });
		userPhone.set('0611111111');
	});

	async function openOptions() {
		render(NotifModal);
		notifModalState.set('subscribed');
		await fireEvent.click(await screen.findByText(new RegExp(m.notifOptions())));
		return waitFor(() => {
			const el = document.querySelector<HTMLInputElement>('.digest-toggle');
			expect(el).not.toBeNull();
			return el!;
		});
	}

	it('is collapsed behind Options until asked for', async () => {
		const get = vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });
		render(NotifModal);
		notifModalState.set('subscribed');
		expect(await screen.findByText(new RegExp(m.notifOptions()))).toBeInTheDocument();
		expect(document.querySelector('.digest-toggle')).toBeNull();
		expect(get).not.toHaveBeenCalled();
	});

	it('shows the saved choice and saves a change', async () => {
		vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });
		const set = vi.spyOn(api.preferences, 'set').mockImplementation(async (_p, prefs) => prefs);
		const toggle = await openOptions();
		expect(toggle.checked).toBe(true);
		expect(document.querySelector('.digest-pref')?.textContent).toContain(m.digestToggleHint({ hour: '8' }));

		await fireEvent.click(toggle);
		await waitFor(() => expect(set).toHaveBeenCalledWith('0611111111', { morning_digest: false }));
		expect(toggle.checked).toBe(false);
	});

	it('goes back and says so when saving fails', async () => {
		vi.spyOn(api.preferences, 'get').mockResolvedValue({ morning_digest: true });
		vi.spyOn(api.preferences, 'set').mockRejectedValue(new Error('offline'));
		const toggle = await openOptions();
		await fireEvent.click(toggle);
		await waitFor(() => expect(document.querySelector('.digest-err')?.textContent).toBe(m.digestSaveErr()));
		expect(toggle.checked).toBe(true);
	});

	it('has no Options when the server sends no digest', async () => {
		config.set({ siteName: 'Go-Stop', returnDelayHours: 2, morningDigestHour: -1 });
		render(NotifModal);
		notifModalState.set('subscribed');
		await screen.findByText(/Envoyer un test|Send a test/);
		expect(screen.queryByText(new RegExp(m.notifOptions()))).toBeNull();
	});
});
