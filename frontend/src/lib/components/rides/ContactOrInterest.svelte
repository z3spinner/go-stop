<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { get } from 'svelte/store';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { pushState, updateBellState } from '$lib/pwa';
	import { openNotifModal } from '$lib/notifModal';
	import { m } from '$lib/paraglide/messages';
	import type { PublicRide, Ride } from '$lib/types';

	let { ride, contactPhone }: { ride: PublicRide | Ride; contactPhone?: string } = $props();

	const storedInterest = () => (browser ? localStorage.getItem(`interest_${ride.ID}`) : null);
	let pending = $state(!!storedInterest());
	let stateMsg = $state('');
	let busy = $state(false);

	async function express() {
		if (busy) return;
		busy = true;
		stateMsg = '';
		let phone = get(userPhone);
		if (!phone && browser) phone = window.prompt(m.labelPhone()) ?? '';
		if (!phone) { busy = false; return; }
		try {
			const res = await api.interests.express(ride.ID, phone, get(userName) || undefined);
			if (browser) localStorage.setItem(`interest_${ride.ID}`, res.id);
			pending = true;
			stateMsg = m.interestSent();
			if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		} catch (e) {
			stateMsg = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}

	async function cancel() {
		if (busy) return;
		const id = storedInterest();
		if (!id) return;
		busy = true;
		stateMsg = '';
		try {
			await api.interests.cancel(id, get(userPhone));
			if (browser) localStorage.removeItem(`interest_${ride.ID}`);
			pending = false;
			stateMsg = m.requestCancelled();
		} catch (e) {
			stateMsg = e instanceof Error ? e.message : String(e);
		} finally {
			busy = false;
		}
	}
</script>

{#if contactPhone}
	<div class="contact-revealed">
		<span class="contact-revealed-label">{m.contactRevealed()}:</span>
		<a href="tel:{contactPhone}">📞 {contactPhone}</a>
	</div>
{:else if pending}
	<div class="interest-pending-row flex items-center gap-2">
		<span class="interest-pending-label text-sm text-gray-600">{m.interestPending()}</span>
		<button type="button" class="btn-interest btn-interest-resend" data-ride-id={ride.ID} disabled={busy} onclick={express}>{m.btnResend()}</button>
		<button type="button" class="btn-interest-cancel btn-ghost-inline" data-ride-id={ride.ID} disabled={busy} onclick={cancel}>{m.btnCancelRequest()}</button>
		<span class="interest-state" id="int-state-{ride.ID}">{stateMsg}</span>
	</div>
{:else}
	<button type="button" class="btn-interest" data-ride-id={ride.ID} disabled={busy} onclick={express}>{m.btnInterest()}</button>
	<span class="interest-state" id="int-state-{ride.ID}">{stateMsg}</span>
{/if}
