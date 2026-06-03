<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	import RideForm from '$lib/components/rides/RideForm.svelte';
	import { openNotifModal } from '$lib/notifModal';
	import { pushState } from '$lib/pwa';
	import { get } from 'svelte/store';
	import { m } from '$lib/paraglide/messages';

	let destinations = $state<string[]>([]);
	onMount(async () => { try { destinations = await api.destinations.list(); } catch { destinations = []; } });

	function posted() {
		if (get(pushState) !== 'subscribed') openNotifModal(get(pushState));
		goto('/my-rides');
	}
</script>

<h2 class="mb-3 text-xl font-semibold">{m.postRideTitle()}</h2>
<RideForm {destinations} onposted={posted} />
