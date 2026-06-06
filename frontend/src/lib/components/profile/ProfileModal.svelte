<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import * as Dialog from '$lib/components/ui/dialog';
	import { profileModalState } from '$lib/profileModal';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';

	let onComplete = $derived($profileModalState);
	let open = $derived(onComplete !== null);

	let name = $state(get(userName));
	let phone = $state(get(userPhone));

	const complete = $derived(name.trim().length > 0 && phone.trim().length > 0);

	$effect(() => {
		if (open) {
			name = get(userName);
			phone = get(userPhone);
		}
	});

	function close() {
		profileModalState.set(null);
	}
	function onOpenChange(v: boolean) {
		if (!v) close();
	}

	function saveAndContinue() {
		if (!complete) return;
		userName.set(name.trim());
		userPhone.set(normalizePhone(phone));
		const cb = onComplete;
		close();
		cb?.();
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title>{m.profileRequiredTitle()}</Dialog.Title>
		</Dialog.Header>
		<p class="text-sm text-gray-600">{m.profileRequiredBody()}</p>
		<div class="mt-1 flex flex-col gap-2">
			<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={name} /></label>
			<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={phone} /></label>
		</div>
		<div class="mt-3 flex gap-2">
			<button type="button" id="btn-profile-save" class="btn btn-primary" disabled={!complete} onclick={saveAndContinue}>{m.btnSaveContinue()}</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
