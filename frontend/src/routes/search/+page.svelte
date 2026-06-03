<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { lastOrigin, lastDestination, userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import RideCard from '$lib/components/rides/RideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, RideSearchParams } from '$lib/types';

	let destinations = $state<string[]>([]);
	let origin = $state('');
	let destination = $state('');
	let search_date = $state('');
	let search_time = $state('');
	let submitted = $state(false);
	let fwd = $state<PublicRide[]>([]);
	let rev = $state<PublicRide[]>([]);
	let contacts = $state<Map<string, string>>(new Map());

	function fromUrl() {
		const sp = page.url.searchParams;
		origin = sp.get('origin') ?? '';
		destination = sp.get('destination') ?? '';
		search_date = sp.get('search_date') ?? '';
		search_time = sp.get('search_time') ?? '';
		const dep = sp.get('departure_at');
		if (dep) { // split a UTC instant into local date + time for the inputs
			const d = new Date(dep);
			const p = (n: number) => String(n).padStart(2, '0');
			search_date = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
			search_time = `${p(d.getHours())}:${p(d.getMinutes())}`;
		}
	}

	async function run() {
		if (!origin || !destination) return;
		submitted = true;
		lastOrigin.set(origin);
		lastDestination.set(destination);

		const params: RideSearchParams = { origin, destination };
		const url = new URLSearchParams({ origin, destination });
		if (search_date && search_time) {
			const iso = new Date(`${search_date}T${search_time}`).toISOString();
			params.departure_at = iso;
			url.set('departure_at', iso);
		} else if (search_date) {
			params.search_date = search_date; url.set('search_date', search_date);
		} else if (search_time) {
			params.search_time = search_time; url.set('search_time', search_time);
		}
		goto(`/search?${url.toString()}`, { replaceState: true, keepFocus: true, noScroll: true });

		const revParams: RideSearchParams = { ...params, origin: destination, destination: origin };
		const [a, b] = await Promise.all([
			api.rides.list(params).catch(() => []),
			api.rides.list(revParams).catch(() => [])
		]);
		fwd = a as PublicRide[];
		rev = b as PublicRide[];
		contacts = await loadAcceptedContacts(get(userPhone));
	}

	function submit(e: SubmitEvent) { e.preventDefault(); run(); }
	function notify(o: string, d: string) {
		const u = new URLSearchParams({ origin: o, destination: d });
		if (search_date && search_time) u.set('departure_at', new Date(`${search_date}T${search_time}`).toISOString());
		goto(`/post-request?${u.toString()}`);
	}

	onMount(async () => {
		try { destinations = await api.destinations.list(); } catch { destinations = []; }
		fromUrl();
		if (!origin && !destination) { origin = get(lastOrigin); destination = get(lastDestination); }
		if (origin && destination) run();
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.findTitle()}</h2>
<form id="search-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={origin} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destination} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<div class="search-datetime-row flex gap-2">
		<label>{m.labelSearchDate()}<input name="search_date" type="date" bind:value={search_date} /></label>
		<label>{m.labelSearchTime()}<input name="search_time" type="time" bind:value={search_time} /></label>
	</div>
	<button type="submit" class="btn btn-primary">{m.btnSearch()}</button>
</form>

{#if submitted}
	<div id="results" class="results-grid mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
		{#each [{ list: fwd, o: origin, d: destination }, { list: rev, o: destination, d: origin }] as col}
			<div class="results-col">
				<div class="results-col-header font-semibold" translate="no">{col.o} <span class="route-arrow">→</span> {col.d}</div>
				{#if col.list.length === 0}
					<div class="col-empty">
						<p>{m.noRidesCol()}</p>
						<button type="button" class="btn-notify-route col-notify underline" data-from={col.o} data-to={col.d} onclick={() => notify(col.o, col.d)}>{m.btnNotifyRoute()}</button>
					</div>
				{:else}
					<div class="flex flex-col gap-2">
						{#each col.list as r}<RideCard ride={r} contactPhone={contacts.get(r.ID)} />{/each}
						<button type="button" class="btn-notify-route col-notify underline" data-from={col.o} data-to={col.d} onclick={() => notify(col.o, col.d)}>{m.btnNotifyRoute()}</button>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
