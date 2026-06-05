<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import { untrack } from 'svelte';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { normalizePhone } from '$lib/utils';
	import ProfileFields from '$lib/components/ProfileFields.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility, PostRequestBody } from '$lib/types';

	let {
		origin = '', destination = '', departureAt = '', destinations = [], onposted
	}: { origin?: string; destination?: string; departureAt?: string; destinations?: string[]; onposted?: (phone: string) => void } = $props();

	type Mode = 'time' | 'day' | 'daily' | 'anytime';
	let mode = $state<Mode>('time');
	let searcher_name = $state(get(userName));
	let phone = $state(get(userPhone));
	let originV = $state(untrack(() => origin));
	let destinationV = $state(untrack(() => destination));
	// Prefill date/time from a passed RFC3339 departureAt (search "notify" deep link).
	let alert_date = $state(untrack(() => (departureAt ? departureAt.slice(0, 10) : '')));
	let alert_time = $state(untrack(() => (departureAt ? new Date(departureAt).toTimeString().slice(0, 5) : '')));
	let flexibility = $state<Flexibility>(30);
	let err = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		err = '';
		const ph = normalizePhone(phone);
		userName.set(searcher_name);
		userPhone.set(ph);
		const body: PostRequestBody = { searcher_name, phone: ph, origin: originV, destination: destinationV };
		if (mode === 'time') {
			if (alert_date && alert_time) { body.departure_at = new Date(`${alert_date}T${alert_time}`).toISOString(); body.flexibility = flexibility; }
			else if (alert_date) body.departure_date = alert_date;
		} else if (mode === 'day') {
			if (alert_date) body.departure_date = alert_date;
		} else if (mode === 'daily') {
			if (alert_time) { body.departure_time = alert_time; body.flexibility = flexibility; }
		} // anytime → nothing
		try {
			await api.requests.post(body);
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
	const modes: { key: Mode; label: () => string }[] = [
		{ key: 'time', label: m.alertModeTime }, { key: 'day', label: m.alertModeDay },
		{ key: 'daily', label: m.alertModeDaily }, { key: 'anytime', label: m.alertModeAnytime }
	];
</script>

<form id="notify-form" onsubmit={submit} class="flex flex-col gap-3">
	<ProfileFields bind:name={searcher_name} bind:phone nameField="searcher_name" />
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={originV} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destinationV} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>

	<div id="alert-mode-btns" class="flex flex-wrap gap-2" role="group">
		{#each modes as mo}
			<button type="button" class="btn-mode" class:active={mode === mo.key} data-mode={mo.key} onclick={() => (mode = mo.key)}>{mo.label()}</button>
		{/each}
	</div>

	{#if mode !== 'anytime'}
		<div id="alert-time-fields" class="flex flex-col gap-2">
			<div class="search-datetime-row flex gap-2">
				{#if mode !== 'daily'}<label>{m.labelSearchDate()}<input name="alert_date" type="date" bind:value={alert_date} /></label>{/if}
				{#if mode !== 'day'}<label>{m.labelSearchTime()}<input name="alert_time" type="time" bind:value={alert_time} /></label>{/if}
			</div>
			{#if mode === 'time' || mode === 'daily'}
				<label>{m.labelFlex()}
					<select bind:value={flexibility}>
						<option value={0}>{m.flexExact()}</option>
						<option value={30}>{m.flex30()}</option>
						<option value={60}>{m.flex60()}</option>
					</select>
				</label>
			{/if}
		</div>
	{/if}

	<button type="submit" class="btn btn-primary">{m.notifEnable()}</button>
	{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
</form>
