<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { formatTime } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide } from '$lib/types';

	const id = page.params.id!;
	let ride = $state<PublicRide | null>(null);
	let done = $state(false);
	let busy = $state(false);
	let err = $state('');

	onMount(async () => {
		// Context only — the question still works if the ride can't be fetched.
		try { ride = await api.rides.get(id); } catch { /* ignore */ }
	});

	async function answer(taken: boolean) {
		if (busy) return;
		busy = true;
		err = '';
		try {
			await api.rides.feedback(id, get(userPhone), taken);
			done = true;
		} catch (e) {
			err = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{m.feedbackTitle()} · Go Stop Saillans!</title></svelte:head>

<section class="feedback-screen mx-auto flex max-w-md flex-col items-center py-8 text-center">
	{#if done}
		<div class="feedback-done text-3xl font-bold text-green-700">{m.feedbackThanks()}</div>
		<button type="button" class="btn btn-secondary mt-8" onclick={() => goto('/')}>{m.btnBack()}</button>
	{:else}
		{#if ride}
			<div class="feedback-route text-xl font-semibold" translate="no">{ride.Origin} <span class="route-arrow">→</span> {ride.Destination}</div>
			<div class="feedback-when mt-1 text-sm text-gray-500">{formatTime(ride.DepartureAt)}</div>
		{/if}

		<!-- The answer buttons read as full statements, so no separate question line. -->
		<div class="feedback-actions mt-8 flex w-full flex-col gap-3">
			<button type="button" class="feedback-yes btn btn-primary" disabled={busy} onclick={() => answer(true)}>{m.feedbackYes()}</button>
			<button type="button" class="feedback-no btn btn-secondary" disabled={busy} onclick={() => answer(false)}>{m.feedbackNo()}</button>
		</div>

		{#if err}<div class="feedback-err mt-4 text-red-600">{err}</div>{/if}
	{/if}
</section>
