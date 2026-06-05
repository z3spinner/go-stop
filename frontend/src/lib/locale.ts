// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	getLocale,
	setLocale,
	locales,
	baseLocale,
	overwriteGetLocale,
	overwriteSetLocale,
	type Locale
} from '$lib/paraglide/runtime';

const KEY = 'lang';

// True in a real browser and in the jsdom test environment, false during SSR /
// prerender. We guard on the actual presence of `localStorage` rather than
// SvelteKit's `$app/environment` `browser` flag, because that flag is compiled
// to `false` under Vitest and would otherwise disable the strategy in tests.
function hasStorage(): boolean {
	return typeof localStorage !== 'undefined';
}

function read(): Locale | undefined {
	if (!hasStorage()) return undefined;
	const v = localStorage.getItem(KEY);
	return v && (locales as readonly string[]).includes(v) ? (v as Locale) : undefined;
}

// Wires the active locale to localStorage["lang"], falling back to baseLocale ("fr").
// Must be called once on the client before the first getLocale() (see +layout.svelte).
//
// Note: this uses overwriteGetLocale/overwriteSetLocale rather than the
// defineCustomClientStrategy('custom-lang', …) path. Both are exported by the
// installed Paraglide runtime, but the custom-strategy chain consults the
// configured fallbacks (preferredLanguage → baseLocale) when "lang" is unset
// or invalid, so an invalid value would resolve to the navigator language
// instead of baseLocale. The overwrite* path resolves read() ?? baseLocale
// directly, which is the contracted behavior (invalid/unset → "fr").
export function registerLangStrategy(): void {
	if (!hasStorage()) return;
	overwriteGetLocale(() => read() ?? baseLocale);
	overwriteSetLocale((l) => {
		localStorage.setItem(KEY, l);
		// Reload so all message functions pick up the new locale. Guarded so the
		// jsdom test environment (where reload is unimplemented) is unaffected.
		if (typeof location !== 'undefined' && typeof location.reload === 'function') {
			location.reload();
		}
	});
}

export { getLocale, setLocale, locales, baseLocale };
export type { Locale };
