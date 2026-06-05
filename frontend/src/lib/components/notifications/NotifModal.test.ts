// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import NotifModal from './NotifModal.svelte';
import { notifModalState } from '$lib/notifModal';

beforeEach(() => notifModalState.set(null));

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
