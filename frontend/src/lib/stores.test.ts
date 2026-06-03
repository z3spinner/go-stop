import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { persisted } from 'svelte-persisted-store';
import { userName, userPhone, lastOrigin, lastDestination, rawString } from './stores';

describe('stores', () => {
	beforeEach(() => localStorage.clear());

	it('persists userName to the user_name key as a RAW string (legacy-compatible)', () => {
		userName.set('Marie');
		// raw value, NOT JSON-encoded — matches the legacy app + existing user data
		expect(localStorage.getItem('user_name')).toBe('Marie');
		expect(get(userName)).toBe('Marie');
	});

	it('persists userPhone to the user_phone key as a RAW string', () => {
		userPhone.set('0644000001');
		expect(localStorage.getItem('user_phone')).toBe('0644000001');
	});

	it('defaults last-search fields to empty strings', () => {
		expect(get(lastOrigin)).toBe('');
		expect(get(lastDestination)).toBe('');
	});

	it('hydrates from RAW legacy data (the common case)', () => {
		localStorage.setItem('legacy_raw', 'Zeno');
		expect(get(persisted('legacy_raw', '', { serializer: rawString }))).toBe('Zeno');
	});

	it('tolerates JSON-quoted data left by the briefly-broken app', () => {
		localStorage.setItem('legacy_quoted', JSON.stringify('Zeno'));
		expect(get(persisted('legacy_quoted', '', { serializer: rawString }))).toBe('Zeno');
	});
});
