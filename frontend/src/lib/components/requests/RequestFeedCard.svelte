<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatTime, formatDate, flexLabel } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRequest } from '$lib/types';

	// A searcher's open request (phone-stripped). Same 4-mode display as AlertCard,
	// but the action is driver-facing: "I can drive this" prefills Post a Ride.
	let { request }: { request: PublicRequest } = $props();

	const ZERO = '0001-01-01T00:00:00Z';
	const hasDate = $derived(request.Date !== ZERO && request.Date?.slice(0, 4) !== '0001');
	const hasTime = $derived(request.DepartureAt !== ZERO && request.DepartureAt?.slice(0, 4) !== '0001');
	const isDaily = $derived(hasTime && request.DepartureAt.slice(0, 10) === '1970-01-01');

	function drive() {
		const u = new URLSearchParams({ origin: request.Origin, destination: request.Destination });
		// Only a concrete one-off date+time can prefill the departure; daily/day/anytime
		// leave it to the driver (no specific instant to seed).
		if (hasTime && !isDaily) u.set('departure_at', request.DepartureAt);
		goto(`/post-ride?${u.toString()}`);
	}
</script>

<div class="card card-compact rounded border p-3" id="req-feed-{request.ID}">
	<div class="card-route font-medium" translate="no">{request.Origin} <span class="route-arrow">→</span> {request.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		{#if request.SearcherName}<span>{request.SearcherName}</span>{/if}
		{#if !hasDate && !hasTime}
			<span class="tag tag-anytime">{m.alertAnytimeLabel()}</span>
		{:else if isDaily}
			<span class="tag tag-daily">{new Date(request.DepartureAt).toISOString().slice(11, 16)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{:else if hasDate && !hasTime}
			<span class="tag">{formatDate(request.Date)}</span>
		{:else}
			<span>{formatTime(request.DepartureAt)}</span>
			<span class="tag">{flexLabel(request.Flexibility)}</span>
		{/if}
	</div>
	<button type="button" class="btn-drive-this btn btn-secondary" data-origin={request.Origin} data-dest={request.Destination} onclick={drive}>{m.btnDriveThis()}</button>
</div>
