<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import { get } from 'svelte/store';
	import { userPhone } from '$lib/stores';
	import { pushState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { openA2HS } from '$lib/a2hs';
	import { config } from '$lib/config';
	import { api } from '$lib/api';
	import {
		DEFAULT_RETENTION_DAYS,
		inboxIcon,
		inboxItems,
		inboxRoute,
		inboxTitle,
		isPastTrip,
		markInboxItemsRead,
		refreshInbox,
		deleteInboxItem,
		restoreInboxItem
	} from '$lib/inbox';
	import SwipeToDelete from '$lib/components/notifications/SwipeToDelete.svelte';
	import type { InboxItem } from '$lib/types';
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
	let days = $derived($config.notificationRetentionDays || DEFAULT_RETENTION_DAYS);
	let pushOn = $derived($pushState === 'subscribed');

	// "Notification deleted · Undo" bar for the most recent deletion.
	const UNDO_MS = 6000;
	let undo = $state<{ id: string; phone: string } | null>(null);
	let undoTimer: ReturnType<typeof setTimeout> | undefined;

	function remove(item: InboxItem) {
		const p = phone;
		deleteInboxItem(p, item);
		clearTimeout(undoTimer);
		undo = { id: item.id, phone: p };
		undoTimer = setTimeout(() => (undo = null), UNDO_MS);
	}

	async function undoDelete() {
		if (!undo) return;
		const { id, phone: p } = undo;
		clearTimeout(undoTimer);
		undo = null;
		await restoreInboxItem(p, id);
	}

	onDestroy(() => clearTimeout(undoTimer));

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

	// Morning digest switch: on by default, saved per phone. Hidden when the
	// server doesn't send one (MORNING_DIGEST_HOUR=none, or an older server).
	let digestHour = $derived($config.morningDigestHour ?? -1);
	let digestOn = $state<boolean | null>(null);
	let digestErr = $state('');
	$effect(() => {
		const p = phone;
		digestOn = null;
		if (!p) return;
		api.preferences
			.get(p)
			.then((r) => {
				if (untrack(() => phone) === p) digestOn = r.morning_digest;
			})
			.catch(() => {});
	});
	async function toggleDigest(e: Event) {
		const on = (e.currentTarget as HTMLInputElement).checked;
		const prev = digestOn;
		digestOn = on;
		digestErr = '';
		try {
			await api.preferences.set(phone, { morning_digest: on });
		} catch {
			digestOn = prev;
			digestErr = m.digestSaveErr();
		}
	}

	// A notification becomes read only when it is tapped (here, or as a push) —
	// never just because it is on screen. Navigation to its link continues.
	function open(item: InboxItem) {
		if (!item.read) markInboxItemsRead(phone, [item.id]);
	}
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
	{#if digestHour >= 0 && digestOn !== null}
		<label class="digest-pref card mb-3 flex items-start gap-3 rounded border p-3">
			<input type="checkbox" class="digest-toggle mt-1 h-4 w-4 shrink-0" checked={digestOn} onchange={toggleDigest} />
			<span class="flex flex-col gap-0.5">
				<span class="font-medium">☀️ {m.digestToggleLabel()}</span>
				<span class="text-sm text-gray-600">{m.digestToggleHint({ hour: String(digestHour) })}</span>
				{#if digestErr}<span class="digest-err text-sm text-red-600">{digestErr}</span>{/if}
			</span>
		</label>
	{/if}

	<div id="inbox-list" class="flex flex-col gap-2">
		{#if loaded && items.length === 0}
			<p class="empty text-gray-500">{m.inboxEmpty()}</p>
		{:else}
			{#each items as item (item.id)}
				{@const unread = !item.read}
				<SwipeToDelete ondelete={() => remove(item)} deleteLabel={m.inboxDeleteAria()} actionLabel={m.inboxDelete()}>
					<a class="inbox-item card rounded border p-3" class:inbox-unread={unread} href={item.url || '/'} data-kind={item.kind} data-id={item.id} draggable="false" onclick={() => open(item)}>
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
				</SwipeToDelete>
			{/each}
		{/if}
	</div>

	{#if items.length > 0}
		<p class="inbox-swipe-hint mt-3 text-xs text-gray-500">{m.inboxSwipeHint()}</p>
	{/if}
	<p class="inbox-retention mt-4 text-xs text-gray-500">{m.inboxRetention({ days })}</p>
{/if}

{#if undo}
	<div class="inbox-undo fixed left-1/2 z-50 flex -translate-x-1/2 items-center gap-4 rounded-lg bg-gray-900 px-4 py-3 text-sm text-white shadow-lg" style="bottom: calc(1rem + env(safe-area-inset-bottom, 0px))" role="status">
		<span>{m.inboxDeleted()}</span>
		<button type="button" class="btn-inbox-undo font-semibold text-green-300 underline" onclick={undoDelete}>{m.inboxUndo()}</button>
	</div>
{/if}

<style>
	/* The swipe hint is for touch screens; mouse users get the delete button. */
	@media (hover: hover) {
		.inbox-swipe-hint {
			display: none;
		}
	}
</style>
