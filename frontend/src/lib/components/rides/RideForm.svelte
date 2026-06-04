<script lang="ts">
	import { get } from 'svelte/store';
	import { untrack } from 'svelte';
	import { api } from '$lib/api';
	import { userName, userPhone } from '$lib/stores';
	import { config } from '$lib/config';
	import { defaultDeparture, normalizePhone } from '$lib/utils';
	import ProfileFields from '$lib/components/ProfileFields.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility } from '$lib/types';

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
			await api.rides.post({
				driver_name, phone: ph, origin, destination,
				departure_at: new Date(departure_at).toISOString(), flexibility
			});
			if (isReturn && return_departure_at) {
				await api.rides.post({
					driver_name, phone: ph, origin: destination, destination: origin,
					departure_at: new Date(return_departure_at).toISOString(), flexibility: return_flexibility
				});
			}
			onposted?.(ph);
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
</script>

<form id="ride-form" onsubmit={submit} class="flex flex-col gap-3">
	<ProfileFields bind:name={driver_name} bind:phone nameField="driver_name" />
	<label>{m.labelFrom()}<input name="origin" list="dests-from" required bind:value={origin} /></label>
	<label>{m.labelTo()}<input name="destination" list="dests-to" required bind:value={destination} /></label>
	<datalist id="dests-from">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<datalist id="dests-to">{#each destinations as d}<option value={d}></option>{/each}</datalist>
	<label>{m.labelDatetime()}<input name="departure_at" type="datetime-local" step="300" required bind:value={departure_at} /></label>
	<label>{m.labelFlex()}
		<select bind:value={flexibility}>
			<option value={0}>{m.flexExact()}</option>
			<option value={30}>{m.flex30()}</option>
			<option value={60}>{m.flex60()}</option>
		</select>
	</label>

	<div class="trip-type-toggle" role="group" aria-label={m.tripTypeLabel()}>
		<button id="btn-oneway" type="button" class="trip-type-btn" class:active={!isReturn} onclick={() => toggleReturn(false)}>{m.tripOneWay()}</button>
		<button id="btn-return" type="button" class="trip-type-btn" class:active={isReturn} onclick={() => toggleReturn(true)}>{m.tripReturn()}</button>
	</div>

	{#if isReturn}
		<fieldset id="return-section" class="return-section flex flex-col gap-2 rounded border p-2">
			<legend>{m.returnSection()}</legend>
			<label>{m.labelReturnTime()}<input name="return_departure_at" type="datetime-local" step="300" bind:value={return_departure_at} required={isReturn} /></label>
			<label>{m.labelReturnFlex()}
				<select bind:value={return_flexibility}>
					<option value={0}>{m.flexExact()}</option>
					<option value={30}>{m.flex30()}</option>
					<option value={60}>{m.flex60()}</option>
				</select>
			</label>
		</fieldset>
	{/if}

	<button type="submit" class="btn btn-primary">{m.btnPostRide()}</button>
	{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
</form>
