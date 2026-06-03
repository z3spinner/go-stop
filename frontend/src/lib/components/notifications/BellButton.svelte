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
	<button type="button" class="btn-bell" class:bell-enabled={subscribed} aria-label="Notifications" data-notif-state={state} onclick={click}>🔔</button>
	{#if !subscribed}<button type="button" class="bell-activate-label" onclick={click}>{m.btnActivate()}</button>{/if}
{/if}
