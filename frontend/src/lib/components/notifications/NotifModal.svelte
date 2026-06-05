<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import * as Dialog from '$lib/components/ui/dialog';
	import { notifModalState } from '$lib/notifModal';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { trySubscribePush, updateBellState, sendTestPush, reconfigurePush } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';

	let modalState = $derived($notifModalState);
	let open = $derived(modalState !== null);
	let err = $state('');
	let busy = $state(false);
	let feedback = $state('');

	// Notifications must be tied to a name (for display) and a phone (for ride
	// matching), so the user has to provide both before we can subscribe.
	const profileComplete = $derived($userName.trim().length > 0 && $userPhone.trim().length > 0);

	function close() {
		notifModalState.set(null);
		feedback = '';
		err = '';
	}
	function onOpenChange(v: boolean) {
		if (!v) close();
	}

	async function enable() {
		err = '';
		if (!profileComplete) return; // button is disabled in this case; guard anyway
		userName.set($userName.trim());
		userPhone.set(normalizePhone($userPhone));
		const phone = get(userPhone);
		const perm = await Notification.requestPermission();
		if (perm === 'granted') {
			await trySubscribePush(phone);
			await updateBellState(phone);
			close();
		} else {
			notifModalState.set('denied');
		}
	}

	async function test() {
		busy = true;
		feedback = '';
		const n = await sendTestPush(get(userPhone));
		feedback = n > 0 ? m.notifTestSent({ count: n }) : m.notifTestNoDevice();
		busy = false;
	}

	async function reconfigure() {
		busy = true;
		feedback = '';
		const ok = await reconfigurePush(get(userPhone));
		feedback = ok ? m.notifReconfigured() : m.notifReconfigureFailed();
		busy = false;
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.notifTitle()}</Dialog.Title>
		</Dialog.Header>

		{#if modalState === 'subscribed' || modalState === 'granted'}
			<p class="text-sm text-gray-600">{m.notifEnabled()}</p>
			<div class="mt-1 flex flex-col gap-2">
				<button type="button" class="btn-test-notif btn btn-primary" disabled={busy} onclick={test}>{m.btnTestNotif()}</button>
				<button type="button" class="btn-reconfigure-notif btn btn-secondary" disabled={busy} onclick={reconfigure}>{m.btnReconfigureNotif()}</button>
			</div>
			{#if feedback}<p class="notif-feedback mt-2 text-sm text-green-700">{feedback}</p>{/if}
		{:else if modalState === 'denied'}
			<p class="text-sm text-gray-600">{m.notifDeniedTip()}</p>
		{:else}
			<p class="text-sm text-gray-600">{m.notifBody()}</p>
			<div class="mt-1 flex flex-col gap-2">
				<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={$userName} /></label>
				<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={$userPhone} /></label>
			</div>
			<div class="mt-3 flex gap-2">
				<button type="button" id="btn-notif-modal-enable" class="btn btn-primary" disabled={!profileComplete} onclick={enable}>{m.notifEnable()}</button>
				<button type="button" id="btn-notif-modal-skip" class="btn btn-secondary" onclick={close}>{m.notifSkip()}</button>
			</div>
			{#if err}<div class="mt-2 text-red-600">{err}</div>{/if}
		{/if}
	</Dialog.Content>
</Dialog.Root>
