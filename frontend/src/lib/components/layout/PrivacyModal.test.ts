// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import PrivacyModal from './PrivacyModal.svelte';

describe('PrivacyModal', () => {
	it('renders the privacy heading and a close button when open', async () => {
		const onclose = vi.fn();
		render(PrivacyModal, { props: { open: true, onclose } });
		expect(screen.getByText(/Confidentialité|Privacy/)).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /Fermer|Close/ })).toBeInTheDocument();
	});

	it('renders nothing when closed', async () => {
		const onclose = vi.fn();
		render(PrivacyModal, { props: { open: false, onclose } });
		expect(screen.queryByRole('dialog')).toBeNull();
	});
});
