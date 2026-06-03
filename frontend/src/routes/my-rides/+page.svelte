<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import MyRideCard from '$lib/components/rides/MyRideCard.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Ride } from '$lib/types';

	let phone = $state(get(userPhone));
	let rides = $state<Ride[]>([]);
	let loaded = $state(false);

	async function load(e?: SubmitEvent) {
		e?.preventDefault();
		userPhone.set(phone);
		try { rides = (await api.rides.list({}, phone)) as Ride[]; } catch { rides = []; }
		loaded = true;
	}
	onMount(async () => { if (phone) { await tick(); load(); } });
</script>

<h2 class="mb-3 text-xl font-semibold">{m.myRidesTitle()}</h2>
<form id="my-rides-form" onsubmit={load} class="mb-4 flex items-end gap-2">
	<label class="grow">{m.labelPhoneCheck()}<input name="phone" type="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnShowRides()}</button>
</form>

<div id="my-rides-list" class="flex flex-col gap-3">
	{#if loaded && rides.length === 0}
		<p class="empty text-gray-500">{m.noMyRides()}</p>
	{:else}
		{#each rides as r (r.ID)}<MyRideCard ride={r} {phone} />{/each}
	{/if}
</div>
