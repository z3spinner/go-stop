<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { get } from 'svelte/store';
	import { untrack } from 'svelte';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { config } from '$lib/config';
	import { defaultDeparture, normalizePhone, formatDate } from '$lib/utils';
	import ProfileFields from '$lib/components/ProfileFields.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility } from '$lib/types';
	import { expandOffsets, shiftDaysIso, type Frequency } from '$lib/recurrence';
	import PlaceCombobox from '$lib/components/forms/PlaceCombobox.svelte';
	import FlexibilitySelect from '$lib/components/forms/FlexibilitySelect.svelte';
	import NumberStepper from '$lib/components/ui/number-stepper/NumberStepper.svelte';
	import * as Select from '$lib/components/ui/select';

	// origin/destination/departureAt seed the form (e.g. "I can drive this" from a
	// requested ride). departureAt is a datetime-local string; falls back to default.
	let {
		destinations = [],
		onposted,
		origin: initialOrigin = '',
		destination: initialDestination = '',
		departureAt: initialDeparture = ''
	}: {
		destinations?: string[];
		onposted?: (phone: string) => void;
		origin?: string;
		destination?: string;
		departureAt?: string;
	} = $props();

	let driver_name = $state(get(userName));
	let phone = $state(get(userPhone));
	// untrack: seed once from the props (deep-link prefill), then own the state.
	let origin = $state(untrack(() => initialOrigin));
	let destination = $state(untrack(() => initialDestination));
	let departure_at = $state(untrack(() => initialDeparture) || defaultDeparture());
	let flexibility = $state<Flexibility>(30);
	let isReturn = $state(false);
	let return_departure_at = $state('');
	let return_flexibility = $state<Flexibility>(30);
	let frequency = $state<Frequency>('none');
	let repeatCount = $state(4);
	let count = $derived(Math.min(14, Math.max(1, Math.floor(Number(repeatCount) || 1))));
	let offsets = $derived(expandOffsets(new Date(departure_at), frequency, frequency === 'none' ? 1 : count));
	let summary = $derived.by(() => {
		if (frequency === 'none' || offsets.length === 0) return '';
		if (isNaN(new Date(departure_at).getTime())) return '';
		const first = formatDate(shiftDaysIso(departure_at, offsets[0]));
		const last = formatDate(shiftDaysIso(departure_at, offsets[offsets.length - 1]));
		return m.repeatSummary({ count: offsets.length, first, last });
	});
	let err = $state('');

	function toggleReturn(on: boolean) {
		isReturn = on;
		if (on && !return_departure_at) {
			if (departure_at) {
				const d = new Date(departure_at);
				d.setHours(d.getHours() + $config.returnDelayHours);
				const p = (n: number) => String(n).padStart(2, '0');
				return_departure_at = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
			} else {
				return_departure_at = defaultDeparture();
			}
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		err = '';
		const ph = normalizePhone(phone);
		userName.set(driver_name);
		userPhone.set(ph);
		try {
			for (const off of offsets) {
				await api.rides.post({
					driver_name, phone: ph, origin, destination,
					departure_at: shiftDaysIso(departure_at, off), flexibility
				});
				if (isReturn && return_departure_at) {
					await api.rides.post({
						driver_name, phone: ph, origin: destination, destination: origin,
						departure_at: shiftDaysIso(return_departure_at, off), flexibility: return_flexibility
					});
				}
			}
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
</script>

<form id="ride-form" onsubmit={submit} class="flex flex-col gap-3">
	<ProfileFields bind:name={driver_name} bind:phone nameField="driver_name" />
	<label>{m.labelFrom()}<PlaceCombobox name="origin" required items={destinations} bind:value={origin} /></label>
	<label>{m.labelTo()}<PlaceCombobox name="destination" required items={destinations} bind:value={destination} /></label>
	<label>{m.labelDatetime()}<input name="departure_at" type="datetime-local" step="300" required bind:value={departure_at} /></label>
	<label>{m.labelFlex()}<FlexibilitySelect bind:value={flexibility} /></label>

	<div class="trip-type-toggle" role="group" aria-label={m.tripTypeLabel()}>
		<button id="btn-oneway" type="button" class="trip-type-btn" class:active={!isReturn} onclick={() => toggleReturn(false)}>{m.tripOneWay()}</button>
		<button id="btn-return" type="button" class="trip-type-btn" class:active={isReturn} onclick={() => toggleReturn(true)}>{m.tripReturn()}</button>
	</div>

	{#if isReturn}
		<fieldset id="return-section" class="return-section flex flex-col gap-2 rounded border p-2">
			<legend>{m.returnSection()}</legend>
			<label>{m.labelReturnTime()}<input name="return_departure_at" type="datetime-local" step="300" bind:value={return_departure_at} required={isReturn} /></label>
			<label>{m.labelReturnFlex()}<FlexibilitySelect bind:value={return_flexibility} /></label>
		</fieldset>
	{/if}

	<label>{m.repeatLabel()}
		<Select.Root type="single" value={frequency} onValueChange={(v) => (frequency = v as typeof frequency)}>
			<Select.Trigger class="w-full">
				{frequency === 'none' ? m.repeatNone() : frequency === 'daily' ? m.repeatDaily() : frequency === 'weekdays' ? m.repeatWeekdays() : m.repeatWeekly()}
			</Select.Trigger>
			<Select.Content>
				<Select.Item value="none" label={m.repeatNone()}>{m.repeatNone()}</Select.Item>
				<Select.Item value="daily" label={m.repeatDaily()}>{m.repeatDaily()}</Select.Item>
				<Select.Item value="weekdays" label={m.repeatWeekdays()}>{m.repeatWeekdays()}</Select.Item>
				<Select.Item value="weekly" label={m.repeatWeekly()}>{m.repeatWeekly()}</Select.Item>
			</Select.Content>
		</Select.Root>
	</label>
	{#if frequency !== 'none'}
		<label>{m.repeatCountLabel()}<NumberStepper bind:value={repeatCount} min={1} max={14} /></label>
		{#if summary}<p class="text-sm text-gray-600">{summary}</p>{/if}
	{/if}
	<button type="submit" class="btn btn-primary">{m.btnPostRide()}</button>
	{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
</form>
