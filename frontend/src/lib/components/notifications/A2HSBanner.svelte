<script lang="ts">
	import { browser } from '$app/environment';
	import { isIOSBrowser } from '$lib/pwa';
	import { openA2HS } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';

	let dismissed = $state(browser ? localStorage.getItem('a2hs_dismissed') === '1' : true);
	let show = $derived(browser && isIOSBrowser() && !dismissed);

	function dismiss() {
		if (browser) localStorage.setItem('a2hs_dismissed', '1');
		dismissed = true;
	}
</script>

{#if show}
	<div id="a2hs-banner" class="a2hs-banner mx-auto flex max-w-xl items-center gap-2 rounded bg-blue-50 p-2">
		<span>🔔 {m.a2hsTitle()}</span>
		<button type="button" id="a2hs-banner-open" class="a2hs-banner-cta underline" onclick={openA2HS}>{m.a2hsHint()}</button>
		<button type="button" id="a2hs-banner-dismiss" class="a2hs-banner-dismiss ml-auto" aria-label="Dismiss" onclick={dismiss}>✕</button>
	</div>
{/if}
