<script lang="ts">
	import { get } from 'svelte/store';
	import { pushState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { openA2HS } from '$lib/a2hs';
	import { m } from '$lib/paraglide/messages';

	let state = $derived($pushState);
	let isIos = $derived(state === 'ios-browser');
	let subscribed = $derived(state === 'subscribed');

	function click() {
		if (isIos) openA2HS();
		else openNotifModal(get(pushState));
	}
</script>

{#if isIos}
	<button type="button" class="bell-activate-label" onclick={openA2HS}>{m.a2hsHint()}</button>
{:else}
	<button type="button" class="btn-bell" class:bell-enabled={subscribed} aria-label="Notifications" data-notif-state={state} onclick={click}><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg></button>
	{#if !subscribed}<button type="button" class="bell-activate-label" onclick={click}>{m.btnActivate()}</button>{/if}
{/if}
