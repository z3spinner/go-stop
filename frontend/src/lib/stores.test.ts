import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { userName, userPhone, lastOrigin, lastDestination } from './stores';

describe('stores', () => {
	beforeEach(() => localStorage.clear());

	it('persists userName to the user_name key', () => {
		userName.set('Marie');
		expect(localStorage.getItem('user_name')).toBe(JSON.stringify('Marie'));
		expect(get(userName)).toBe('Marie');
	});

	it('persists userPhone to the user_phone key', () => {
		userPhone.set('0644000001');
		expect(localStorage.getItem('user_phone')).toBe(JSON.stringify('0644000001'));
	});

	it('defaults last-search fields to empty strings', () => {
		expect(get(lastOrigin)).toBe('');
		expect(get(lastDestination)).toBe('');
	});
});
