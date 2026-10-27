<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { pushState } from '#lib/pwa.js';
	import { openNotifModal } from '#lib/notifModal.js';
	import { openA2HS } from '#lib/a2hs.js';
	import { inboxUnread, badgeText } from '#lib/inbox.js';
	import { userPhone } from '#lib/stores.js';
	import { m } from '#lib/paraglide/messages.js';

	let state = $derived($pushState);
	let isIos = $derived(state === 'ios-browser');
	let subscribed = $derived(state === 'subscribed');
	// With a phone in the profile the bell opens the inbox (push settings live
	// there); without one there is no inbox yet, so it offers push setup as before.
	let hasPhone = $derived($userPhone.trim().length > 0);
	let badge = $derived(hasPhone ? badgeText($inboxUnread) : '');

	function setupPush() {
		if (isIos) openA2HS();
		else openNotifModal(get(pushState));
	}
	function click() {
		if (hasPhone) goto('/notifications');
		else setupPush();
	}
</script>

{#if isIos && !hasPhone}
	<button type="button" class="bell-activate-label" onclick={openA2HS}>{m.a2hsHint()}</button>
{:else}
	<button type="button" class="btn-bell relative" class:bell-enabled={subscribed} aria-label={badge ? m.inboxUnreadAria({ count: $inboxUnread }) : m.inboxTitle()} data-notif-state={state} onclick={click}><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>{#if badge}<span class="bell-badge" aria-hidden="true">{badge}</span>{/if}</button>
	{#if isIos}
		<button type="button" class="bell-activate-label" onclick={openA2HS}>{m.a2hsHint()}</button>
	{:else if !subscribed}
		<button type="button" class="bell-activate-label" onclick={setupPush}>{m.btnActivate()}</button>
	{/if}
{/if}
