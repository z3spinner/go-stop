<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import '../app.css';
	import '../legacy.css'; // ported live-site styles; targets the preserved semantic class names
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { browser } from '$app/environment';
	import { registerLangStrategy } from '$lib/locale';
	import { loadConfig } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { get } from 'svelte/store';
	import { updateBellState, pollForNotifications } from '$lib/pwa';
	import TopBar from '$lib/components/layout/TopBar.svelte';
	import PrivacyModal from '$lib/components/layout/PrivacyModal.svelte';
	import A2HSBanner from '$lib/components/notifications/A2HSBanner.svelte';
	import A2HSModal from '$lib/components/notifications/A2HSModal.svelte';
	import PollToastHost from '$lib/components/notifications/PollToast.svelte';
	import NotifModal from '$lib/components/notifications/NotifModal.svelte';
	import ProfileModal from '$lib/components/profile/ProfileModal.svelte';
	import PullToRefresh from '$lib/components/layout/PullToRefresh.svelte';
	import { m } from '$lib/paraglide/messages';

	let { children } = $props();
	let showPrivacy = $state(false);
	let isHome = $derived(page.url.pathname === '/');

	// Bumping this remounts the page subtree, re-running each page's onMount fetch.
	let refreshNonce = $state(0);
	async function refresh() {
		await invalidateAll(); // re-run load() for the pages that use it
		refreshNonce++; // remount to re-run onMount-based fetches
	}

	function back() {
		// Always return to the home hub. history.back() is unreliable here because
		// history.length counts entries from before the app loaded (other sites in
		// the same tab), so it could navigate out of the app instead of home.
		goto('/');
	}

	if (browser) registerLangStrategy();

	onMount(() => {
		if ('serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(() => {});
		loadConfig();
		const phone = get(userPhone);
		updateBellState(phone);
		// No notification prompt on first launch: it used to fire here before the
		// user could orient or pick a language, and a language change reloads the
		// page (locale.ts) — discarding the prompt and forcing the bell to be
		// re-activated. Notifications are offered contextually instead (after
		// posting a ride/request or expressing interest) and via the bell, which
		// shows an "Activate" label until subscribed.
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
	<TopBar onprivacy={() => (showPrivacy = true)} />
</header>

<PullToRefresh onrefresh={refresh} />

<div id="app" class="mx-auto max-w-xl p-3">
	{#key refreshNonce}
		{@render children()}
	{/key}
</div>

<footer id="app-footer" class="mx-auto max-w-xl p-3 text-center text-sm text-gray-500">
	<button type="button" class="btn-footer-privacy underline" onclick={() => (showPrivacy = true)}>{m.footerPrivacy()}</button>
	<span> · </span>
	<a class="btn-footer-about underline" href="/about">{m.aboutTitle()}</a>
	<span> · </span>
	<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
</footer>

<A2HSBanner />
<A2HSModal />
<PollToastHost />
<NotifModal />
<ProfileModal />
<PrivacyModal open={showPrivacy} onclose={() => (showPrivacy = false)} />
