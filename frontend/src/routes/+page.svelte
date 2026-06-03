<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { config } from '$lib/config';
	import { userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import RideCard from '$lib/components/rides/RideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Stats } from '$lib/types';

	let rides = $state<PublicRide[]>([]);
	let contacts = $state<Map<string, string>>(new Map());
	let stats = $state<Stats | null>(null);
	let pendingBadge = $state(0);

	onMount(async () => {
		const phone = get(userPhone);
		contacts = await loadAcceptedContacts(phone);
		try { rides = (await api.rides.list()) as PublicRide[]; } catch { rides = []; }
		try { stats = await api.stats.get(); } catch { stats = null; }
		// pending-interest badge: count pending interests across my own rides
		if (phone) {
			try {
				const mine = (await api.rides.list({}, phone)) as { ID: string }[];
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
	<h2 class="home-feed-title mb-2 font-semibold">{m.homeFeedTitle()}</h2>
	{#if rides.length === 0}
		<p class="home-feed-empty text-gray-500">{m.noActiveRides()}</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#each rides as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} />{/each}
		</div>
	{/if}
</section>

<section id="home-stats" class="mt-5">
	{#if stats && stats.total_confirmed > 0}
		<div class="stats-widget rounded border p-3">
			<div class="stats-widget-title font-semibold">{m.statsAllTime({ n: stats.total_confirmed })}</div>
			{#each stats.top_routes as rt}
				<button type="button" class="stats-row stats-row-btn block w-full text-left" data-origin={rt.Origin} data-dest={rt.Destination}
					onclick={() => goto(`/search?origin=${encodeURIComponent(rt.Origin)}&destination=${encodeURIComponent(rt.Destination)}`)}>
					{rt.Origin} → {rt.Destination} <span class="stats-count">{m.statsRouteCount({ n: rt.Count })}</span>
				</button>
			{/each}
			<a class="btn-all-stats underline" href="/stats">{m.btnAllStats()}</a>
		</div>
	{/if}
</section>
