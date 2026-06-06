// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import ProfileModal from './ProfileModal.svelte';
import { profileModalState, openProfileModal } from '$lib/profileModal';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => {
	localStorage.clear();
	userName.set('');
	userPhone.set('');
	profileModalState.set(null);
});

describe('ProfileModal', () => {
	it('saves name + phone and runs the continuation, then closes', async () => {
		const onComplete = vi.fn();
		render(ProfileModal);
		openProfileModal(onComplete);

		const name = await screen.findByRole('textbox', { name: /name|nom/i });
		await fireEvent.input(name, { target: { value: 'Marie' } });
		const phone = document.querySelector('input[type="tel"]') as HTMLInputElement;
		await fireEvent.input(phone, { target: { value: '0612345678' } });

		await fireEvent.click(screen.getByRole('button', { name: /continue|continuer/i }));

		await vi.waitFor(() => {
			expect(onComplete).toHaveBeenCalledTimes(1);
			expect(get(userName)).toBe('Marie');
			expect(get(userPhone)).toBe('0612345678');
			expect(get(profileModalState)).toBeNull();
		});
	});

	it('disables the save button until both fields are filled', async () => {
		render(ProfileModal);
		openProfileModal(vi.fn());
		const btn = (await screen.findByRole('button', { name: /continue|continuer/i })) as HTMLButtonElement;
		expect(btn.disabled).toBe(true);
	});

	it('does not run the continuation when dismissed without saving', async () => {
		const onComplete = vi.fn();
		render(ProfileModal);
		openProfileModal(onComplete);
		await screen.findByRole('button', { name: /continue|continuer/i });

		// dismiss without saving
		profileModalState.set(null);

		await vi.waitFor(() => expect(get(profileModalState)).toBeNull());
		expect(onComplete).not.toHaveBeenCalled();
	});
});
