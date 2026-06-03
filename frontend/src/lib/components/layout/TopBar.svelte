<script lang="ts">
	import LangPicker from './LangPicker.svelte';
	import BellButton from '$lib/components/notifications/BellButton.svelte';
	import { userName } from '$lib/stores';

	let { onabout, onprivacy }: { onabout?: () => void; onprivacy?: () => void } = $props();
	let hasProfile = $derived($userName.length > 0);
</script>

<div class="controls flex items-center gap-2">
	<LangPicker />
	<div class="controls-icons ml-auto flex items-center gap-2">
		<a id="btn-me" href="/me" class="btn-me-icon" class:me-icon-set={hasProfile} aria-label="Profile" title="Me">
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
		</a>
		<button type="button" class="btn-privacy" aria-label="About" onclick={onabout}>ⓘ</button>
		<BellButton />
	</div>
</div>
