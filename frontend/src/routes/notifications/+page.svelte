<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
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
		markAllInboxRead,
		refreshInbox
	} from '$lib/inbox';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { InboxItem } from '$lib/types';

	const phone = get(userPhone).trim();
	// A snapshot, not the live store: opening the page marks everything read on
	// the server (clearing the bell), but items stay highlighted for this visit
	// so the user can still see what was new.
	let items = $state<InboxItem[]>([]);
	let loaded = $state(false);
	let days = $derived($config.notificationRetentionDays || DEFAULT_RETENTION_DAYS);
	let pushOn = $derived($pushState === 'subscribed');

	function settings() {
		const s = get(pushState);
		if (s === 'ios-browser') openA2HS();
		else openNotifModal(s);
	}

	onMount(async () => {
		if (!phone) {
			loaded = true;
			return;
		}
		await refreshInbox(phone, { force: true });
		items = get(inboxItems);
		loaded = true;
		if (items.some((i) => !i.read)) markAllInboxRead(phone);
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
				<a class="inbox-item card rounded border p-3" class:inbox-unread={!item.read} href={item.url || '/'} data-kind={item.kind}>
					<span class="inbox-icon" aria-hidden="true">{inboxIcon(item)}</span>
					<span class="flex min-w-0 flex-col gap-0.5">
						<span class="inbox-title font-medium">{inboxTitle(item)}</span>
						<span class="inbox-route text-sm text-gray-700" translate="no">{inboxRoute(item)}</span>
						<span class="inbox-meta flex flex-wrap items-center gap-2 text-xs text-gray-500">
							<span>{formatTime(item.created_at)}</span>
							{#if isPastTrip(item)}<span class="tag">{m.inboxPast()}</span>{/if}
						</span>
					</span>
				</a>
			{/each}
		{/if}
	</div>

	<p class="inbox-retention mt-4 text-xs text-gray-500">{m.inboxRetention({ days })}</p>
{/if}
