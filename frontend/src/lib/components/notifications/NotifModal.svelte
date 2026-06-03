<script lang="ts">
	import { get } from 'svelte/store';
	import { notifModalState } from '$lib/notifModal';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { trySubscribePush, updateBellState } from '$lib/pwa';
	import { m } from '$lib/paraglide/messages';

	let modalState = $derived($notifModalState);
	let err = $state('');

	// Notifications must be tied to a name (for display) and a phone (for ride
	// matching), so the user has to provide both before we can subscribe.
	const profileComplete = $derived($userName.trim().length > 0 && $userPhone.trim().length > 0);

	function close() { notifModalState.set(null); }

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
</script>

<svelte:window onkeydown={(e) => { if (modalState && e.key === 'Escape') close(); }} />

{#if modalState}
	<div class="modal-overlay fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onclick={close} role="presentation">
		<div class="modal w-full max-w-sm rounded bg-white p-5" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex={-1}>
			<h3 class="mb-2 text-lg font-semibold">{m.notifTitle()}</h3>
			{#if modalState === 'subscribed' || modalState === 'granted'}
				<p>{m.notifEnabled()}</p>
				<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
			{:else if modalState === 'denied'}
				<p>{m.notifDeniedTip()}</p>
				<button type="button" class="mt-3 rounded border px-3 py-1" onclick={close}>{m.privacyClose()}</button>
			{:else}
				<p>{m.notifBody()}</p>
				<div class="mt-3 flex flex-col gap-2">
					<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={$userName} /></label>
					<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={$userPhone} /></label>
				</div>
				<div class="mt-3 flex gap-2">
					<button type="button" id="btn-notif-modal-enable" class="btn btn-primary" disabled={!profileComplete} onclick={enable}>{m.notifEnable()}</button>
					<button type="button" id="btn-notif-modal-skip" class="btn btn-secondary" onclick={close}>{m.notifSkip()}</button>
				</div>
				{#if err}<div class="mt-2 text-red-600">{err}</div>{/if}
			{/if}
		</div>
	</div>
{/if}
