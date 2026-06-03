<script lang="ts">
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { MyInterest } from '$lib/types';

	let { interest }: { interest: MyInterest } = $props();
	const accepted = $derived(interest.status === 'accepted' || interest.status === 'driver_shared');
</script>

<div class="card rounded border p-3" id="req-card-{interest.id}">
	<div class="card-route font-medium" translate="no">{interest.origin} → {interest.destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(interest.departure_at)}</span>
		<span>{interest.driver_name}</span>
		{#if accepted}
			<span class="tag tag-accepted">{m.reqStatusAccepted()}</span>
		{:else}
			<span class="tag">{m.reqStatusPending()}</span>
		{/if}
	</div>
	{#if accepted}
		<a class="btn-contact-link" href="/interests/{interest.id}">{m.contactRevealed()} →</a>
	{:else}
		<span class="interest-pending-label text-sm text-gray-600">{m.interestPending()}</span>
	{/if}
</div>
