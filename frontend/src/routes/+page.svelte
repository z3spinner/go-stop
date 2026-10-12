<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { config } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import RideCard from '$lib/components/rides/RideCard.svelte';
	import RequestFeedCard from '$lib/components/requests/RequestFeedCard.svelte';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, PublicRequest } from '$lib/types';

	let rides = $state<PublicRide[]>([]);
	let requests = $state<PublicRequest[]>([]);
	let contacts = $state<Map<string, string>>(new Map());
	let pendingBadge = $state(0);
	let myIds = $state<Set<string>>(new Set());

	// Swipe ↔ tabs: `tab` is the source of truth. Tapping a tab scrolls the panel
	// into view (with a short guard so the in-flight smooth-scroll doesn't fight
	// the scroll listener); swiping updates `tab` from the scroll position.
	let tab = $state('available');
	let swipeEl = $state<HTMLDivElement | null>(null);
	let snapping = false;
	let snapTimer: ReturnType<typeof setTimeout>;

	function snapTo(value: string) {
		if (!swipeEl) return;
		snapping = true;
		swipeEl.scrollTo({ left: (value === 'requested' ? 1 : 0) * swipeEl.clientWidth, behavior: 'smooth' });
		clearTimeout(snapTimer);
		snapTimer = setTimeout(() => (snapping = false), 450);
	}
	function onScroll() {
		if (snapping || !swipeEl) return;
		tab = Math.round(swipeEl.scrollLeft / swipeEl.clientWidth) === 1 ? 'requested' : 'available';
	}

	onMount(async () => {
		const phone = get(userPhone);
		contacts = await loadAcceptedContacts(phone);
		try { rides = (await api.rides.list()) as PublicRide[]; } catch { rides = []; }
		try { requests = await api.requests.listActive(); } catch { requests = []; }
		// pending-interest badge: count pending interests across my own rides
		if (phone) {
			try {
				const mine = (await api.rides.list({}, phone)) as { ID: string }[];
				myIds = new Set(mine.map((r) => r.ID));
				let n = 0;
				for (const r of mine) {
					const ints = await api.rides.listInterests(r.ID, phone);
					n += ints.filter((i) => i.status === 'pending').length;
				}
				pendingBadge = n;
			} catch { pendingBadge = 0; }
		}
	});
</script>

<div class="hero text-center">
	<h1 class="text-2xl font-bold">{$config.siteName}</h1>
	<p class="tagline text-gray-600">{m.tagline()}</p>
	<div class="mt-4 flex flex-col gap-2">
		<button type="button" class="btn btn-primary" onclick={() => goto('/post-ride')}>{m.btnDriver()}</button>
		<button type="button" class="btn btn-secondary" onclick={() => goto('/search')}>{m.btnSearcher()}</button>
	</div>
	<div class="ghost-row mt-3 flex items-center justify-center gap-2 text-sm">
		<button type="button" class="btn-ghost-inline" onclick={() => goto('/me')}>{m.btnMe()}</button>
		<span class="ghost-sep">·</span>
		<button type="button" class="btn-ghost-inline relative" onclick={() => goto('/my-rides')}>
			{m.btnMyRides()}{#if pendingBadge > 0}<span class="interest-badge ml-1 rounded-full bg-red-500 px-1 text-white">{pendingBadge}</span>{/if}
		</button>
		<span class="ghost-sep">·</span>
		<button type="button" class="btn-ghost-inline" onclick={() => goto('/my-searches')}>{m.btnMySearches()}</button>
	</div>
</div>

<section id="home-feed" class="mt-5">
	<div class="feed-tabs">
		<Tabs.Root bind:value={tab} onValueChange={snapTo}>
			<Tabs.List class="grid w-full grid-cols-2">
				<Tabs.Trigger value="available">{m.tabAvailable()}</Tabs.Trigger>
				<Tabs.Trigger value="requested">{m.tabRequested()}</Tabs.Trigger>
			</Tabs.List>
		</Tabs.Root>
	</div>

	<div class="feed-swipe mt-2" bind:this={swipeEl} onscroll={onScroll}>
		<div class="feed-panel" role="tabpanel" aria-label={m.tabAvailable()}>
			{#if rides.length === 0}
				<p class="home-feed-empty text-gray-500">{m.noActiveRides()}</p>
			{:else}
				<div class="flex flex-col gap-2">
					{#each rides as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} isOwn={myIds.has(r.ID)} />{/each}
				</div>
			{/if}
		</div>
		<div class="feed-panel" role="tabpanel" aria-label={m.tabRequested()}>
			{#if requests.length === 0}
				<p class="home-feed-empty text-gray-500">{m.noRequests()}</p>
			{:else}
				<div class="flex flex-col gap-2">
					{#each requests as rq (rq.ID)}<RequestFeedCard request={rq} />{/each}
				</div>
			{/if}
		</div>
	</div>
</section>

<style>
	/* Restyle the shadcn Tabs to match the app's green segmented control
	   (.trip-type-toggle): flat, white track, green active with white text.
	   Scoped via .feed-tabs; :global reaches the shadcn elements, and being
	   unlayered these rules win over Tailwind's layered utilities. */
	.feed-tabs :global([data-slot='tabs-list']) {
		background: #fff;
		border: 1px solid var(--gray-300, #d1d5db);
		border-radius: var(--radius, 8px);
		padding: 0;
		gap: 0;
		height: auto;
	}
	.feed-tabs :global([data-slot='tabs-trigger']) {
		border-radius: 0;
		padding: 9px;
		font-size: 0.9rem;
		font-weight: 500;
		color: var(--gray-600, #4b5563);
		background: transparent;
		box-shadow: none;
		transition: background 0.15s;
	}
	.feed-tabs :global([data-slot='tabs-trigger']:last-child) {
		border-left: 1px solid var(--gray-300, #d1d5db);
	}
	.feed-tabs :global([data-slot='tabs-trigger']:hover:not([data-state='active'])) {
		background: var(--gray-100, #f3f4f6);
	}
	.feed-tabs :global([data-slot='tabs-trigger'][data-state='active']) {
		background: var(--blue, #28a836);
		color: #fff;
		font-weight: 600;
		box-shadow: none;
	}

	/* Two full-width panels side by side; horizontal swipe snaps between them. */
	.feed-swipe {
		display: flex;
		overflow-x: auto;
		scroll-snap-type: x mandatory;
		scrollbar-width: none; /* Firefox */
		-webkit-overflow-scrolling: touch;
	}
	.feed-swipe::-webkit-scrollbar {
		display: none; /* Chrome/Safari */
	}
	.feed-panel {
		flex: 0 0 100%;
		min-width: 0;
		scroll-snap-align: start;
	}
</style>
