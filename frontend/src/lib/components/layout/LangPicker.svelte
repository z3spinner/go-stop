<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { getLocale, setLocale, locales } from '$lib/locale';

	const FLAGS: Record<string, string> = { fr: '🇫🇷', en: '🇬🇧', es: '🇪🇸', it: '🇮🇹', de: '🇩🇪', nl: '🇳🇱', el: '🇬🇷' };
	const ORDER = ['fr', 'en', 'es', 'it', 'de', 'nl', 'el'] as const;
	let open = $state(false);
	let current = $derived(getLocale());

	function pick(code: string) {
		open = false;
		setLocale(code as never); // persists to localStorage["lang"] and reloads
	}
</script>

<div class="lang-picker relative">
	<button type="button" class="btn-lang" aria-label="Language" onclick={() => (open = !open)}>
		{FLAGS[current] ?? '🌐'}
	</button>
	<div class="lang-dropdown absolute right-0 z-50 mt-1 rounded border bg-white shadow" class:hidden={!open}>
		{#each ORDER as code}
			<button type="button" class="lang-opt block w-full px-3 py-1 text-left" data-lang={code} onclick={() => pick(code)}>
				{FLAGS[code]} {code.toUpperCase()}
			</button>
		{/each}
	</div>
	<span class="hidden">{locales.length}</span>
</div>
