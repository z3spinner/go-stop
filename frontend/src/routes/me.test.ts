// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Me from './me/+page.svelte';
import { userName, userPhone } from '$lib/stores';

beforeEach(() => { localStorage.clear(); userName.set(''); userPhone.set(''); });

describe('me', () => {
	it('saves to localStorage and reveals #me-saved', async () => {
		const { container } = render(Me);
		await fireEvent.input(container.querySelector('input[name=name]')!, { target: { value: 'Marie' } });
		await fireEvent.input(container.querySelector('input[name=phone]')!, { target: { value: '0644000001' } });
		await fireEvent.submit(container.querySelector('#me-form')!);
		// Stored RAW (legacy-compatible), not JSON-encoded.
		expect(localStorage.getItem('user_name')).toBe('Marie');
		const saved = container.querySelector('#me-saved') as HTMLElement;
		expect(saved.getAttribute('style') ?? '').not.toContain('none');
	});

	it('pre-fills from an existing profile', () => {
		userName.set('Jean'); userPhone.set('0655000002');
		const { container } = render(Me);
		expect((container.querySelector('input[name=name]') as HTMLInputElement).value).toBe('Jean');
		expect((container.querySelector('input[name=phone]') as HTMLInputElement).value).toBe('0655000002');
	});
});
