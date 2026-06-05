// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';
import en from './en.json';
import fr from './fr.json';
import es from './es.json';
import itMessages from './it.json';
import de from './de.json';
import nl from './nl.json';

const files = { en, fr, es, it: itMessages, de, nl };
const keys = (o: Record<string, unknown>) => Object.keys(o).filter((k) => k !== '$schema').sort();

describe('message files', () => {
	it('all locales share the identical key set', () => {
		const base = keys(en);
		for (const [loc, obj] of Object.entries(files)) {
			expect(keys(obj as Record<string, unknown>), `locale ${loc}`).toEqual(base);
		}
	});
	it('has no empty values', () => {
		for (const [loc, obj] of Object.entries(files)) {
			for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
				if (k === '$schema') continue;
				expect(String(v).length, `${loc}.${k}`).toBeGreaterThan(0);
			}
		}
	});
});
