<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import LangPicker from './LangPicker.svelte';
	import BellButton from '$lib/components/notifications/BellButton.svelte';
	import ShareButton from '$lib/components/ShareButton.svelte';
	import { userName } from '$lib/stores';
	import { config } from '$lib/config';
	import { m } from '$lib/paraglide/messages';

	let { onprivacy }: { onprivacy?: () => void } = $props();
	let hasProfile = $derived($userName.length > 0);
	// The logo + share live on the left of the header on the home page only.
	// Every other page shows the back button there instead, so both are hidden.
	// (The ride detail page carries its own share next to the title.)
	let isHome = $derived(page.url.pathname === '/');
</script>

<div class="controls flex flex-1 items-center gap-2">
	{#if isHome}
		<a href="/" class="brand-logo inline-flex items-center" aria-label={$config.siteName} title={$config.siteName}>
			<img src="/logo.svg" alt="" width="26" height="26" />
		</a>
		<ShareButton title={$config.siteName} text={m.tagline()} size={16} />
	{/if}
	<div class="controls-icons ml-auto flex items-center gap-2">
		<LangPicker />
		<a id="btn-me" href="/me" class="btn-me-icon" class:me-icon-set={hasProfile} aria-label="Profile" title="Me">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
		</a>
		<a href="/about" class="btn-privacy" aria-label="About" title="About"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg></a>
		<BellButton />
	</div>
</div>
