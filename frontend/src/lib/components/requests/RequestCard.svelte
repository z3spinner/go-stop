<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { api } from '$lib/api';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { MyInterest } from '$lib/types';

	let {
		interest,
		phone = '',
		oncancelled
	}: { interest: MyInterest; phone?: string; oncancelled?: (id: string) => void } = $props();
	const accepted = $derived(interest.status === 'accepted' || interest.status === 'driver_shared');

	let busy = $state(false);
	let errMsg = $state('');

	async function cancel() {
		if (busy || !phone) return;
		busy = true;
		errMsg = '';
		try {
			await api.interests.cancel(interest.id, phone);
			if (browser) localStorage.removeItem(`interest_${interest.ride_id}`);
			oncancelled?.(interest.id);
		} catch (e) {
			errMsg = e instanceof Error ? e.message : String(e);
			busy = false;
		}
	}
</script>

<div class="card rounded border p-3" id="req-card-{interest.id}">
	<div class="card-route font-medium" translate="no">{interest.origin} <span class="route-arrow">→</span> {interest.destination}</div>
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
		<div class="interest-pending-row flex items-center gap-2">
			<span class="interest-pending-label text-sm text-gray-600">{m.interestPending()}</span>
			<button type="button" class="btn-interest-cancel btn-ghost-inline" data-id={interest.id} disabled={busy} onclick={cancel}>{m.btnCancelRequest()}</button>
		</div>
		{#if errMsg}<span class="interest-state text-sm text-red-600">{errMsg}</span>{/if}
	{/if}
</div>
