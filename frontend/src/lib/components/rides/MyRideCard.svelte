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
	let fbDone = $state(false);
	let confirmingDelete = $state(false);

	const isPast = $derived(new Date(ride.DepartureAt).getTime() < Date.now());
	const showFeedback = $derived(isPast && !ride.FeedbackGiven && !fbDone);

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
	async function feedback(taken: boolean) {
		try { await api.rides.feedback(ride.ID, phone, taken); fbDone = true; } catch { /* ignore */ }
	}
	async function del() {
		try { await api.rides.del(ride.ID, phone); deleted = true; delMsg = m.deleteOk(); }
		catch { delMsg = m.deleteErr(); }
	}
	function requestDelete() {
		// Ask "did someone come along?" only for trips that have actually departed
		// and haven't been answered yet. Future deletes are silent cancellations.
		if (isPast && !ride.FeedbackGiven && !fbDone) { confirmingDelete = true; }
		else { del(); }
	}
	async function deleteWithFeedback(taken: boolean) {
		try { await api.rides.feedback(ride.ID, phone, taken); fbDone = true; } catch { /* best-effort */ }
		confirmingDelete = false;
		await del();
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

	{#if showFeedback}
		<div class="feedback-prompt mt-2" id="fb-{ride.ID}">
			<div class="feedback-question text-sm">{m.feedbackTitle()}</div>
			<div class="feedback-btns flex gap-2">
				<button type="button" class="btn-fb-yes" data-id={ride.ID} data-phone={phone} onclick={() => feedback(true)}>{m.feedbackYes()}</button>
				<button type="button" class="btn-fb-no" data-id={ride.ID} data-phone={phone} onclick={() => feedback(false)}>{m.feedbackNo()}</button>
			</div>
		</div>
	{:else if fbDone}
		<div class="feedback-thanks text-sm text-green-600">{m.feedbackThanks()}</div>
	{/if}

	{#if confirmingDelete}
		<div class="delete-confirm mt-2" id="del-confirm-{ride.ID}">
			<div class="delete-confirm-q text-sm">{m.deleteAskCameAlong()}</div>
			<div class="delete-confirm-btns flex gap-2">
				<button type="button" class="btn-del-yes" onclick={() => deleteWithFeedback(true)}>{m.feedbackYes()}</button>
				<button type="button" class="btn-del-no" onclick={() => deleteWithFeedback(false)}>{m.feedbackNo()}</button>
				<button type="button" class="btn-del-cancel" onclick={() => (confirmingDelete = false)}>{m.btnCancel()}</button>
			</div>
		</div>
	{:else}
		<button type="button" class="btn btn-danger btn-delete" data-id={ride.ID} data-phone={phone} disabled={deleted} onclick={requestDelete}>{m.btnDelete()}</button>
	{/if}
	<div class="delete-msg" id="msg-{ride.ID}">{delMsg}</div>
</div>
