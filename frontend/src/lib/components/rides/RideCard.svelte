<script lang="ts">
	import ContactOrInterest from './ContactOrInterest.svelte';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Ride } from '$lib/types';

	let {
		ride,
		contactPhone,
		showDriver = true
	}: { ride: PublicRide | Ride; contactPhone?: string; showDriver?: boolean } = $props();

	const interestCount = $derived('InterestCount' in ride ? ride.InterestCount : 0);
</script>

<div class="card card-compact rounded border p-3">
	<div class="card-route font-medium" translate="no">{ride.Origin} → {ride.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(ride.DepartureAt)}</span>
		<span class="tag rounded bg-gray-100 px-1">{flexLabel(ride.Flexibility)}</span>
		{#if showDriver && ride.DriverName}<span>{ride.DriverName}</span>{/if}
		{#if interestCount > 0}<span class="tag tag-interest-count rounded bg-blue-100 px-1">{m.interestCount({ count: interestCount })}</span>{/if}
	</div>
	<ContactOrInterest {ride} {contactPhone} />
</div>
