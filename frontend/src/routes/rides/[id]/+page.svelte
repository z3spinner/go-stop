<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { Ride } from '$lib/types';

	let ride = $state<Ride | null>(null);
	let err = $state('');
	onMount(async () => {
		try { ride = await api.rides.get(page.params.id!); }
		catch (e) { err = e instanceof Error ? e.message : String(e); }
	});
</script>

<h2 class="mb-3 text-xl font-semibold">{m.detailRideTitle()}</h2>
{#if err}
	<p class="error text-red-600">{err}</p>
{:else if ride}
	<div class="card detail-card rounded border p-3">
		<div class="card-route font-medium" translate="no">{ride.Origin} → {ride.Destination}</div>
		<div class="card-meta flex gap-2 text-sm text-gray-600"><span>{formatTime(ride.DepartureAt)}</span><span class="tag">{flexLabel(ride.Flexibility)}</span></div>
		<div class="detail-table mt-2">
			<div>{m.labelDriver()}: {ride.DriverName}</div>
			<div>{m.labelContact()}: <a href="tel:{ride.Phone}">{ride.Phone}</a></div>
		</div>
	</div>
{:else}
	<p>…</p>
{/if}
