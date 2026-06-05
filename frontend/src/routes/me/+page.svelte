<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import { m } from '$lib/paraglide/messages';

	let name = $state(get(userName));
	let phone = $state(get(userPhone));
	let saved = $state(false);

	function submit(e: SubmitEvent) {
		e.preventDefault();
		userName.set(name);
		userPhone.set(normalizePhone(phone));
		saved = true;
		setTimeout(() => (saved = false), 2000);
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.meTitle()}</h2>
<form id="me-form" onsubmit={submit} class="flex flex-col gap-3">
	<label>{m.labelName()}<input name="name" autocomplete="given-name" bind:value={name} /></label>
	<label>{m.labelPhone()}<input name="phone" type="tel" autocomplete="tel" bind:value={phone} /></label>
	<button type="submit" class="btn btn-primary">{m.btnSave()}</button>
	<div id="me-saved" class="section-hint text-green-600" style:display={saved ? 'block' : 'none'}>{m.meSaved()}</div>
</form>
<p class="section-hint text-sm text-gray-500">{m.meHint()}</p>
