<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ContactOrInterest from './ContactOrInterest.svelte';
	import { formatTime, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Ride } from '$lib/types';

	let {
		ride,
		contactPhone,
		showDriver = true,
		isOwn = false
	}: { ride: PublicRide | Ride; contactPhone?: string; showDriver?: boolean; isOwn?: boolean } = $props();

	const interestCount = $derived('InterestCount' in ride ? ride.InterestCount : 0);
</script>

<div class="card card-compact rounded border p-3">
	<a href="/rides/{ride.ID}" class="card-detail-link block text-inherit no-underline" data-ride-id={ride.ID}>
		<div class="card-route font-medium" translate="no">{ride.Origin} <span class="route-arrow">→</span> {ride.Destination}</div>
		<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
			{#if isOwn}<span class="tag tag-your-ride rounded bg-green-100 px-1">{m.yourRide()}</span>{/if}
			<span>{formatTime(ride.DepartureAt)}</span>
			<span class="tag rounded bg-gray-100 px-1">{flexLabel(ride.Flexibility)}</span>
			{#if showDriver && ride.DriverName}<span>{ride.DriverName}</span>{/if}
			{#if interestCount > 0}<span class="tag tag-interest-count rounded bg-blue-100 px-1">{m.interestCount({ count: interestCount })}</span>{/if}
		</div>
	</a>
	{#if isOwn}
		<a href="/my-rides#card-{ride.ID}" class="ride-manage-link inline-block text-sm underline">{m.manageInMyRides()} →</a>
	{:else}
		<ContactOrInterest {ride} {contactPhone} />
	{/if}
</div>
