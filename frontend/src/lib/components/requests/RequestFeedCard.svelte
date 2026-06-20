<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { openProfileModal } from '$lib/profileModal';
	import { normalizePhone } from '$lib/utils';
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

	let busy = $state(false);
	let offered = $state(false);
	let offerError = $state('');
	let syncedPhone = $state('');
	const currentPhone = $derived(normalizePhone($userPhone));
	const shareButtonText = $derived(offered ? m.contactOfferSent() : m.btnShareContact());

	function offerKey(phone: string) {
		return `contact_offer_${encodeURIComponent(phone)}::${encodeURIComponent(request.ID)}`;
	}

	$effect(() => {
		if (!browser || syncedPhone === currentPhone) return;
		syncedPhone = currentPhone;
		offered = currentPhone !== '' && localStorage.getItem(offerKey(currentPhone)) === '1';
	});

	function drive() {
		const u = new URLSearchParams({ origin: request.Origin, destination: request.Destination });
		// Only a concrete one-off date+time can prefill the departure; daily/day/anytime
		// leave it to the driver (no specific instant to seed).
		if (hasTime && !isDaily) u.set('departure_at', request.DepartureAt);
		goto(`/post-ride?${u.toString()}`);
	}

	async function shareContact() {
		if (busy || offered) return;
		const name = get(userName).trim();
		const phone = normalizePhone(get(userPhone));
		if (!name || !phone) {
			openProfileModal(() => shareContact());
			return;
		}
		busy = true;
		offerError = '';
		try {
			await api.requests.offerContact(request.ID, phone, name);
			if (browser) localStorage.setItem(offerKey(phone), '1');
			offered = true;
		} catch (e) {
			offerError = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
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
	<div class="req-actions mt-1.5 flex flex-wrap gap-2">
		<button type="button" class="btn-drive-this" data-origin={request.Origin} data-dest={request.Destination} onclick={drive}>{m.btnDriveThis()}</button>
		<button type="button" class="btn-share-contact" class:shared={offered} data-request-id={request.ID} aria-label={shareButtonText} title={shareButtonText} disabled={busy || offered} onclick={shareContact}>{shareButtonText}</button>
	</div>
	{#if offerError}<span class="offer-state mt-1 text-sm text-gray-600">{offerError}</span>{/if}
</div>

<style>
	/* Compact outline-green button matching the rides panel's "Demander le
	   contact" (.btn-interest), so both feeds read at the same weight. */
	.btn-drive-this {
		padding: 5px 10px;
		font-size: 0.85rem;
		border: 1px solid var(--blue, #28a836);
		border-radius: var(--radius, 8px);
		background: none;
		color: var(--blue, #28a836);
		cursor: pointer;
	}
	.btn-drive-this:hover {
		background: var(--blue, #28a836);
		color: #fff;
	}
	/* Muted outline button for the secondary "share contact" action. */
	.btn-share-contact {
		padding: 5px 10px;
		font-size: 0.85rem;
		border: 1px solid var(--gray-400, #9ca3af);
		border-radius: var(--radius, 8px);
		background: none;
		color: var(--gray-600, #4b5563);
		cursor: pointer;
	}
	.btn-share-contact:hover:not(:disabled),
	.btn-share-contact.shared {
		border-color: var(--blue, #28a836);
		color: var(--blue, #28a836);
	}
	.btn-share-contact:disabled {
		opacity: 0.5;
		cursor: default;
	}
</style>
