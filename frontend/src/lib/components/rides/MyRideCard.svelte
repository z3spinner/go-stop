<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { formatTime, flexLabel } from '$lib/utils';
	import SeekerRow from './SeekerRow.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Ride, PublicRequest, InterestListItem } from '$lib/types';

	let { ride, phone }: { ride: Ride; phone: string } = $props();
	let seekers = $state<PublicRequest[]>([]);
	let interests = $state<InterestListItem[]>([]);
	let deleted = $state(false);
	let delMsg = $state('');
	// The driver's pending answer to "did someone come along?": null until chosen.
	// It is only committed to the server when they hit Delete, so they can change
	// it freely until then.
	let chosen = $state<boolean | null>(null);

	const isPast = $derived(new Date(ride.DepartureAt).getTime() < Date.now());
	// A past ride that hasn't been answered yet must be answered before deletion.
	const needsAnswer = $derived(isPast && !ride.FeedbackGiven);
	const canDelete = $derived(!needsAnswer || chosen !== null);

	onMount(async () => {
		try { seekers = await api.rides.listMatchingRequests(ride.ID, phone); } catch { seekers = []; }
		try { interests = await api.rides.listInterests(ride.ID, phone); } catch { interests = []; }
	});

	async function accept(it: InterestListItem) {
		try {
			const res = await api.interests.accept(it.id, phone);
			interests = interests.map((x) => x.id === it.id ? { ...x, status: 'accepted', searcher_phone: res.searcher_phone } : x);
		} catch { /* surfaced inline below if needed */ }
	}
	async function del() {
		// Commit the (final) feedback choice before deleting, so the answer reflects
		// whatever the driver had selected at the moment they hit Delete.
		if (needsAnswer && chosen !== null) {
			try { await api.rides.feedback(ride.ID, phone, chosen); } catch { /* best-effort */ }
		}
		try { await api.rides.del(ride.ID, phone); deleted = true; delMsg = m.deleteOk(); }
		catch { delMsg = m.deleteErr(); }
	}
	const pendingCount = $derived(interests.filter((i) => i.status === 'pending').length);
	const displayName = (it: InterestListItem) => it.searcher_name?.trim() || m.anonymousSearcher();
</script>

<div class="card rounded border p-3" id="card-{ride.ID}" style:opacity={deleted ? 0.4 : 1}>
	<div class="card-route font-medium" translate="no">{ride.Origin} <span class="route-arrow">→</span> {ride.Destination}</div>
	<div class="card-meta flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span>{formatTime(ride.DepartureAt)}</span>
		<span class="tag">{flexLabel(ride.Flexibility)}</span>
	</div>

	<div class="seekers-section" id="seekers-{ride.ID}">
		{#if seekers.length > 0}
			<div class="seekers-title mt-2 text-sm font-medium">{m.seekersTitle()}</div>
			{#each seekers as s}<SeekerRow request={s} rideId={ride.ID} driverPhone={phone} />{/each}
		{:else}
			<div class="seekers-empty text-sm text-gray-500">{m.noSeekers()}</div>
		{/if}
	</div>

	<div class="interests-section" id="interests-{ride.ID}">
		{#if pendingCount > 0}<div class="interests-title mt-2 text-sm font-medium">{m.pendingInterests({ count: pendingCount })}</div>{/if}
		{#each interests as it}
			<div class="interest-row" id="irow-{it.id}">
				{#if it.status === 'pending'}
					<span class="interest-pending-info">{m.interestPendingName({ name: displayName(it) })}</span>
					<button type="button" class="btn-accept-interest" data-id={it.id} data-phone={phone} onclick={() => accept(it)}>{m.btnAccept()}</button>
				{:else if it.status === 'driver_shared'}
					<span class="interest-accepted">{m.notifSentShort()}</span>
				{:else}
					<span class="interest-accepted">{displayName(it)}{#if it.searcher_phone} — <a href="tel:{it.searcher_phone}">{it.searcher_phone}</a>{/if}</span>
				{/if}
			</div>
		{/each}
	</div>

	<!-- A past ride must be answered before it can be deleted. The choice stays
	     visible (selected option highlighted) and can be changed until Delete is
	     hit, at which point the final choice is committed. -->
	{#if needsAnswer && !deleted}
		<div class="feedback-prompt mt-2" id="fb-{ride.ID}">
			<div class="feedback-question text-sm">{m.feedbackTitle()}</div>
			<div class="feedback-btns">
				<button type="button" class="btn-fb-yes" class:selected={chosen === true} aria-pressed={chosen === true} data-id={ride.ID} data-phone={phone} onclick={() => (chosen = true)}>{m.feedbackYes()}</button>
				<button type="button" class="btn-fb-no" class:selected={chosen === false} aria-pressed={chosen === false} data-id={ride.ID} data-phone={phone} onclick={() => (chosen = false)}>{m.feedbackNo()}</button>
			</div>
		</div>
	{/if}

	<div class="card-actions mt-2 flex gap-2">
		<a class="btn btn-edit" href="/rides/{ride.ID}/edit">{m.btnEdit()}</a>
		<button type="button" class="btn btn-danger btn-delete" data-id={ride.ID} data-phone={phone} disabled={deleted || !canDelete} onclick={del}>{m.btnDelete()}</button>
	</div>
	<div class="delete-msg" id="msg-{ride.ID}">{delMsg}</div>
</div>
