<script lang="ts">
	import '../app.css';
	import '../legacy.css'; // ported live-site styles; targets the preserved semantic class names
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { registerLangStrategy } from '$lib/locale';
	import { loadConfig } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { get } from 'svelte/store';
	import { updateBellState, pollForNotifications, isStandalone, maybeMarkStandalonePrompted } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import TopBar from '$lib/components/layout/TopBar.svelte';
	import AboutModal from '$lib/components/layout/AboutModal.svelte';
	import PrivacyModal from '$lib/components/layout/PrivacyModal.svelte';
	import A2HSBanner from '$lib/components/notifications/A2HSBanner.svelte';
	import A2HSModal from '$lib/components/notifications/A2HSModal.svelte';
	import PollToastHost from '$lib/components/notifications/PollToast.svelte';
	import NotifModal from '$lib/components/notifications/NotifModal.svelte';
	import { m } from '$lib/paraglide/messages';

	let { children } = $props();
	let showAbout = $state(false);
	let showPrivacy = $state(false);
	let isHome = $derived(page.url.pathname === '/');

	function back() {
		if (browser && history.length > 1) history.back();
		else goto('/');
	}

	if (browser) registerLangStrategy();

	onMount(() => {
		if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
		loadConfig();
		const phone = get(userPhone);
		updateBellState(phone);
		if (isStandalone() && maybeMarkStandalonePrompted()) {
			openNotifModal('default');
		}
		const onVis = () => {
			if (document.visibilityState === 'visible') pollForNotifications(get(userPhone));
		};
		document.addEventListener('visibilitychange', onVis);
		return () => document.removeEventListener('visibilitychange', onVis);
	});
</script>

<header class="top-bar mx-auto flex max-w-xl items-center gap-2 p-3" class:page-bar={!isHome}>
	{#if !isHome}
		<button id="back" type="button" class="btn-back" onclick={back}>{m.btnBack()}</button>
	{/if}
	<TopBar onabout={() => (showAbout = true)} onprivacy={() => (showPrivacy = true)} />
</header>

<div id="app" class="mx-auto max-w-xl p-3">
	{@render children()}
</div>

<footer id="app-footer" class="mx-auto max-w-xl p-3 text-center text-sm text-gray-500">
	<button type="button" class="btn-footer-privacy underline" onclick={() => (showPrivacy = true)}>{m.footerPrivacy()}</button>
	<span> · </span>
	<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
</footer>

<A2HSBanner />
<A2HSModal />
<PollToastHost />
<NotifModal />
{#if showAbout}<AboutModal onclose={() => (showAbout = false)} />{/if}
{#if showPrivacy}<PrivacyModal onclose={() => (showPrivacy = false)} />{/if}
