<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import AlertCard from '$lib/components/alerts/AlertCard.svelte';
	import RequestCard from '$lib/components/requests/RequestCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Request, MyInterest } from '$lib/types';

	let phone = $state(get(userPhone));
	let alerts = $state<Request[]>([]);
	let requests = $state<MyInterest[]>([]);
	let loaded = $state(false);

	async function load(e?: SubmitEvent) {
		e?.preventDefault();
		userPhone.set(phone);
		const [a, r] = await Promise.all([
			api.requests.list(phone).catch(() => []),
			api.interests.listMine(phone).catch(() => [])
		]);
		alerts = a; requests = r; loaded = true;
	}
	function seeMatches(o: string, d: string, dep: string) {
		const u = new URLSearchParams({ origin: o, destination: d });
		if (dep) u.set('departure_at', dep);
		goto(`/search?${u.toString()}`);
	}
	onMount(async () => { if (phone) { await tick(); load(); } });
</script>

<h2 class="mb-3 text-xl font-semibold">{m.mySearchesTitle()}</h2>
<form id="my-searches-form" onsubmit={load} class="mb-4 flex items-end gap-2">
	<label class="grow">{m.labelPhoneCheck()}<input name="phone" type="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnShowSearches()}</button>
</form>

<div id="my-searches-content">
	<section>
		<div class="section-label font-semibold">{m.myAlertsTitle()}</div>
		<div id="my-alerts-list" class="flex flex-col gap-2">
			{#if loaded && alerts.length === 0}
				<p class="empty text-gray-500">{m.noMyAlerts()}</p>
			{:else}
				{#each alerts as a (a.ID)}<AlertCard request={a} onseematches={seeMatches} />{/each}
			{/if}
		</div>
	</section>
	<section class="mt-5">
		<div class="section-label font-semibold">{m.myRequestsTitle()}</div>
		<div id="my-requests-list" class="flex flex-col gap-2">
			{#if loaded && requests.length === 0}
				<p class="empty text-gray-500">{m.noMyRequests()}</p>
			{:else}
				{#each requests as r (r.id)}<RequestCard interest={r} />{/each}
			{/if}
		</div>
	</section>
</div>
