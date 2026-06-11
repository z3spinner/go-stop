<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import { userPhone } from '$lib/stores';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility } from '$lib/types';

	const id = page.params.id!;
	let destinations = $state<string[]>([]);
	let origin = $state('');
	let destination = $state('');
	let departure_at = $state('');
	let flexibility = $state<Flexibility>(30);
	let loaded = $state(false);
	let err = $state('');

	// ISO timestamp -> `datetime-local` value (local time, no seconds).
	function isoToLocal(iso: string): string {
		const d = new Date(iso);
		const p = (n: number) => String(n).padStart(2, '0');
		return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
	}

	onMount(async () => {
		try { destinations = await api.destinations.list(); } catch { destinations = []; }
		try {
			const ride = await api.rides.get(id);
			origin = ride.Origin;
			destination = ride.Destination;
			departure_at = isoToLocal(ride.DepartureAt);
			flexibility = ride.Flexibility as Flexibility;
			loaded = true;
		} catch (e) {
			err = e instanceof Error ? e.message : String(e);
		}
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		err = '';
		try {
			await api.rides.update(id, {
				phone: get(userPhone),
				origin,
				destination,
				departure_at: new Date(departure_at).toISOString(),
				flexibility
			});
			goto('/my-rides');
		} catch (ex) {
			err = ex instanceof Error ? ex.message : String(ex);
		}
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.editRideTitle()}</h2>

{#if loaded}
	<form id="edit-ride-form" onsubmit={submit} class="flex flex-col gap-3">
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
		<button type="submit" class="btn btn-primary">{m.btnSaveChanges()}</button>
		{#if err}<div id="err" class="text-red-600">{err}</div>{/if}
	</form>
{:else if err}
	<div class="text-red-600">{err}</div>
{/if}
