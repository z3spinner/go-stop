<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	// Name + phone are seeded from the saved profile. When both are already filled
	// the user has nothing to type, so we hide the inputs and show a compact
	// summary with an inline "Change" — keeping the form short for returning users.
	// `nameField` is the input's `name` attribute (driver_name vs searcher_name).
	let {
		name = $bindable(),
		phone = $bindable(),
		nameField = 'name'
	}: { name: string; phone: string; nameField?: string } = $props();

	let editing = $state(name.trim() === '' || phone.trim() === '');
</script>

{#if editing}
	<label>{m.labelName()}<input name={nameField} required bind:value={name} /></label>
	<label>{m.labelPhone()}<input name="phone" type="tel" required bind:value={phone} /></label>
{:else}
	<div class="profile-summary flex flex-wrap items-center gap-2 text-sm text-gray-600">
		<span><strong>{name}</strong> · {phone}</span>
		<button type="button" class="btn-edit-contact btn-ghost-inline" onclick={() => (editing = true)}>{m.btnEditContact()}</button>
	</div>
{/if}
