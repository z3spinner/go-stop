import { describe, it, expect, vi, afterEach } from 'vitest';
import { normalizePhone, defaultDeparture, localeToBCP47, flexLabel } from './utils';

afterEach(() => vi.useRealTimers());

describe('normalizePhone', () => {
	it('strips spaces, dashes, dots and parens', () => {
		expect(normalizePhone(' (06) 11-00.00 01 ')).toBe('0611000001');
	});
});

describe('localeToBCP47', () => {
	it('maps app locales to Intl locales', () => {
		expect(localeToBCP47('en')).toBe('en-GB');
		expect(localeToBCP47('fr')).toBe('fr-FR');
		expect(localeToBCP47('de')).toBe('de-DE');
	});
});

describe('flexLabel', () => {
	it('maps 0/30/60 to compact labels', () => {
		expect(flexLabel(0)).toMatch(/Exact/i);
		expect(flexLabel(30)).toContain('30');
		expect(flexLabel(60)).toContain('60');
	});
});

describe('defaultDeparture', () => {
	it('returns now + 1h rounded up to 5 min as a local datetime-local string', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2030-12-01T09:02:00'));
		// 09:02 + 1h = 10:02 → rounded up to 10:05
		expect(defaultDeparture()).toBe('2030-12-01T10:05');
	});
});
