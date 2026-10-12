<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { untrack } from 'svelte';
	import { get } from 'svelte/store';
	import { userPhone } from '$lib/stores';
	import { pushState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { openA2HS } from '$lib/a2hs';
	import { config } from '$lib/config';
	import {
		DEFAULT_RETENTION_DAYS,
		inboxIcon,
		inboxItems,
		inboxRoute,
		inboxTitle,
		isPastTrip,
		markInboxItemsRead,
		refreshInbox
	} from '$lib/inbox';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';

	// Follows the profile phone live (it can be entered from the settings modal
	// opened on this page).
	let phone = $derived($userPhone.trim());
	// The live list: items that arrive while the page is open (a push lands and
	// the layout refreshes the store) show up here too.
	let items = $derived($inboxItems);
	let loadedFor = $state<string | null>(null);
	let loaded = $derived(phone !== '' && loadedFor === phone);
	// Ids that were unread when shown during this visit. They stay highlighted
	// after being marked read on the server, so the user can see what was new.
	let shownUnread = $state<Set<string>>(new Set());
	let days = $derived($config.notificationRetentionDays || DEFAULT_RETENTION_DAYS);
	let pushOn = $derived($pushState === 'subscribed');

	function settings() {
		const s = get(pushState);
		if (s === 'ios-browser') openA2HS();
		else openNotifModal(s);
	}

	// (Re)load whenever the phone changes.
	$effect(() => {
		const p = phone;
		if (!p) return;
		refreshInbox(p, { force: true }).then(() => {
			if (untrack(() => phone) === p) loadedFor = p;
		});
	});

	// Mark read exactly the unread items now on screen — never items the page
	// hasn't displayed (a notification arriving later gets its own pass).
	$effect(() => {
		const p = phone;
		const list = items;
		if (!loaded) return;
		const fresh = untrack(() => list.filter((i) => !i.read && !shownUnread.has(i.id)).map((i) => i.id));
		if (fresh.length === 0) return;
		shownUnread = new Set([...untrack(() => shownUnread), ...fresh]);
		markInboxItemsRead(p, fresh);
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.inboxTitle()}</h2>

{#if !phone}
	<p class="inbox-no-phone mb-3 text-gray-600">{m.inboxNoPhone()}</p>
	<a class="btn btn-primary" href="/me">{m.inboxGoToProfile()}</a>
{:else}
	<div class="mb-3 flex flex-wrap items-center gap-2">
		<button type="button" class="btn-inbox-settings btn btn-secondary" onclick={settings}>{m.inboxPushSettings()}</button>
	</div>
	{#if !pushOn}
		<p class="inbox-push-off mb-3 text-sm text-amber-700">{m.inboxPushOff()}</p>
	{/if}

	<div id="inbox-list" class="flex flex-col gap-2">
		{#if loaded && items.length === 0}
			<p class="empty text-gray-500">{m.inboxEmpty()}</p>
		{:else}
			{#each items as item (item.id)}
				{@const unread = !item.read || shownUnread.has(item.id)}
				<a class="inbox-item card rounded border p-3" class:inbox-unread={unread} href={item.url || '/'} data-kind={item.kind}>
					<span class="inbox-icon" aria-hidden="true">{inboxIcon(item)}</span>
					<span class="flex min-w-0 flex-col gap-0.5">
						<span class="inbox-title font-medium">{#if unread}<span class="sr-only">{m.inboxUnreadLabel()}: </span>{/if}{inboxTitle(item)}</span>
						<span class="inbox-route text-sm text-gray-700" translate="no">{inboxRoute(item)}</span>
						<span class="inbox-meta flex flex-wrap items-center gap-2 text-xs text-gray-500">
							<span>{formatTime(item.created_at)}</span>
							{#if item.kind === 'test'}<span class="tag inbox-test-tag">{m.inboxTestTag()}</span>{/if}
							{#if isPastTrip(item)}<span class="tag">{m.inboxPast()}</span>{/if}
						</span>
					</span>
				</a>
			{/each}
		{/if}
	</div>

	<p class="inbox-retention mt-4 text-xs text-gray-500">{m.inboxRetention({ days })}</p>
{/if}
