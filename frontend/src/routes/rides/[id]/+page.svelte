<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { loadAcceptedContacts } from '$lib/contacts';
	import { formatTime, flexLabel } from '$lib/utils';
	import ContactOrInterest from '$lib/components/rides/ContactOrInterest.svelte';
	import ShareButton from '$lib/components/ShareButton.svelte';
	import { config } from '$lib/config';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide } from '$lib/types';

	let ride = $state<PublicRide | null>(null);
	let contactPhone = $state<string | undefined>(undefined);
	let notFound = $state(false);
	let loading = $state(true);

	onMount(async () => {
		const id = page.params.id!;
		try {
			ride = await api.rides.get(id);
			// If the viewer already has an accepted interest for this ride, surface
			// the revealed phone (consistent with the search results).
			contactPhone = (await loadAcceptedContacts(get(userPhone))).get(id);
		} catch {
			notFound = true;
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	{#if ride}
		<title>{ride.Origin} → {ride.Destination} · {$config.siteName}</title>
	{/if}
</svelte:head>

<div class="detail-head mb-3 flex items-center gap-2">
	<h2 class="detail-title text-xl font-semibold">{m.detailRideTitle()}</h2>
	{#if ride}<ShareButton title={`${ride.Origin} → ${ride.Destination}`} text={m.shareRideText()} />{/if}
</div>

{#if loading}
	<p>…</p>
{:else if notFound || !ride}
	<p class="error text-gray-600">{m.detailNotFound()}</p>
{:else}
	<div class="card detail-card rounded border p-3">
		<div class="card-route font-medium" translate="no">{ride.Origin} <span class="route-arrow">→</span> {ride.Destination}</div>
		<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
			<span>{formatTime(ride.DepartureAt)}</span>
			<span class="tag rounded bg-gray-100 px-1">{flexLabel(ride.Flexibility)}</span>
			{#if ride.DriverName}<span>{m.labelDriver()}: {ride.DriverName}</span>{/if}
		</div>
		<ContactOrInterest {ride} {contactPhone} />
	</div>
{/if}

<style>
	/* The global `h2 { margin: 0 0 16px }` (legacy.css) would inflate this flex
	   row and push the share icon below the title's centre; the wrapper's own
	   mb-3 already provides the spacing, so drop the heading's bottom margin. */
	.detail-title {
		margin-bottom: 0;
	}
</style>
