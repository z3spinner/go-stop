<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import '../app.css';
	import '../legacy.css'; // ported live-site styles; targets the preserved semantic class names
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto, refreshAll } from '$app/navigation';
	import { browser } from '$app/env';
	import { registerLangStrategy } from '#lib/locale.js';
	import { loadConfig } from '#lib/config.js';
	import { userPhone } from '#lib/stores.js';
	import { get } from 'svelte/store';
	import { updateBellState } from '#lib/pwa.js';
	import { refreshInbox, markInboxItemsRead, inboxIdFromUrl } from '#lib/inbox.js';
	import TopBar from '#lib/components/layout/TopBar.svelte';
	import PrivacyModal from '#lib/components/layout/PrivacyModal.svelte';
	import A2HSBanner from '#lib/components/notifications/A2HSBanner.svelte';
	import A2HSModal from '#lib/components/notifications/A2HSModal.svelte';
	import NotifModal from '#lib/components/notifications/NotifModal.svelte';
	import ProfileModal from '#lib/components/profile/ProfileModal.svelte';
	import PullToRefresh from '#lib/components/layout/PullToRefresh.svelte';
	import { m } from '#lib/paraglide/messages.js';

	let { children } = $props();
	let showPrivacy = $state(false);
	let isHome = $derived(page.url.pathname === '/');
	let isFlyer = $derived(page.url.pathname === '/flyer');

	// Bumping this remounts the page subtree, re-running each page's onMount fetch.
	let refreshNonce = $state(0);
	async function refresh() {
		await refreshAll(); // re-run load() for the pages that use it
		refreshNonce++; // remount to re-run onMount-based fetches
	}

	function back() {
		// Always return to the home hub. history.back() is unreliable here because
		// history.length counts entries from before the app loaded (other sites in
		// the same tab), so it could navigate out of the app instead of home. A
		// single notification goes back to the list it was opened from.
		goto(page.url.pathname.startsWith('/notifications/') ? '/notifications' : '/');
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
		// Keep the bell's unread badge current: on load and whenever the profile
		// phone changes (debounced — the phone field updates the store per
		// keystroke), when the app comes back to the foreground, and right away
		// when a push arrives while it's open (the service worker posts
		// 'inbox-updated').
		let phoneTimer: ReturnType<typeof setTimeout> | undefined;
		let firstPhone = true;
		const unsubPhone = userPhone.subscribe((p) => {
			clearTimeout(phoneTimer);
			if (firstPhone) {
				firstPhone = false;
				refreshInbox(p.trim(), { force: true });
			} else {
				phoneTimer = setTimeout(() => refreshInbox(p.trim(), { force: true }), 600);
			}
		});
		// Opened by tapping a push (sw.js adds ?n=<inbox id>): that notification
		// has been seen, so mark it read (which stops its reminders) and drop the
		// parameter from the address bar.
		const fromPush = inboxIdFromUrl(new URL(location.href));
		if (fromPush.id) {
			goto(fromPush.clean, { shallow: true, replace: true, state: page.state }).catch(() => {
				/* router not ready: the leftover ?n= is harmless */
			});
			markInboxItemsRead(phone.trim(), [fromPush.id]);
		}
		// Leftover from the old toast polling, replaced by the inbox.
		try {
			localStorage.removeItem('poll_seen');
		} catch {
			/* storage unavailable */
		}
		// Coming back to the app (switching to it, or reopening the installed app
		// from the home screen, which may restore the page without reloading it):
		// always reload so the bell's count matches the home-screen badge. The
		// three events often fire together, so only the first one in 2 s counts.
		let lastResume = 0;
		const onResume = () => {
			if (document.visibilityState !== 'visible') return;
			const now = Date.now();
			if (now - lastResume < 2000) return;
			lastResume = now;
			refreshInbox(get(userPhone).trim(), { force: true });
		};
		const onSwMessage = (e: MessageEvent) => {
			if (e.data?.type === 'inbox-updated') refreshInbox(get(userPhone).trim(), { force: true });
			// A push was tapped while this tab was already open.
			if (e.data?.type === 'inbox-read' && typeof e.data.id === 'string') {
				markInboxItemsRead(get(userPhone).trim(), [e.data.id]);
			}
		};
		document.addEventListener('visibilitychange', onResume);
		window.addEventListener('pageshow', onResume);
		window.addEventListener('focus', onResume);
		navigator.serviceWorker?.addEventListener('message', onSwMessage);
		return () => {
			unsubPhone();
			clearTimeout(phoneTimer);
			document.removeEventListener('visibilitychange', onResume);
			window.removeEventListener('pageshow', onResume);
			window.removeEventListener('focus', onResume);
			navigator.serviceWorker?.removeEventListener('message', onSwMessage);
		};
	});
</script>

<!-- The whole page (header + content + footer) lives inside PullToRefresh so a
     pull drags it down natively; modals/toasts below stay outside (overlays). -->
<PullToRefresh onrefresh={refresh}>
	{#if isFlyer}
		{#key refreshNonce}
			{@render children()}
		{/key}
	{:else}
		<header class="top-bar mx-auto flex max-w-xl items-center gap-2 p-3" class:page-bar={!isHome}>
			{#if !isHome}
				<button id="back" type="button" class="btn-back" onclick={back}>{m.btnBack()}</button>
			{/if}
			<TopBar onprivacy={() => (showPrivacy = true)} />
		</header>

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
			<a class="btn-footer-flyer underline" href="/flyer">{m.flyerMetaTitle()}</a>
			<span> · </span>
			<a class="btn-footer-stats underline" href="/stats">{m.statsPageTitle()}</a>
		</footer>
	{/if}
</PullToRefresh>

<A2HSBanner />
<A2HSModal />
<NotifModal />
<ProfileModal />
<PrivacyModal open={showPrivacy} onclose={() => (showPrivacy = false)} />
