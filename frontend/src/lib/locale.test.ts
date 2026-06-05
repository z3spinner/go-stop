// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, beforeEach } from 'vitest';
import { registerLangStrategy } from './locale';
import { getLocale } from '$lib/paraglide/runtime';

describe('lang persistence', () => {
	beforeEach(() => localStorage.clear());

	it('reads the active locale from localStorage["lang"]', () => {
		localStorage.setItem('lang', 'en');
		registerLangStrategy();
		expect(getLocale()).toBe('en');
	});

	it('falls back to baseLocale (fr) when lang is unset/invalid', () => {
		localStorage.setItem('lang', 'zz');
		registerLangStrategy();
		expect(getLocale()).toBe('fr');
	});
});
