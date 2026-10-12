<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Combobox } from 'bits-ui';

	let {
		value = $bindable(''),
		items = [],
		name = undefined,
		id = undefined,
		placeholder = '',
		required = false,
		disabled = false
	}: {
		value?: string;
		items?: string[];
		name?: string;
		id?: string;
		placeholder?: string;
		required?: boolean;
		disabled?: boolean;
	} = $props();

	let open = $state(false);

	function fold(s: string): string {
		return s
			.normalize('NFD')
			.replace(/\p{Diacritic}/gu, '')
			.toLowerCase()
			.trim();
	}

	let filtered = $derived(
		value.trim() === '' ? items : items.filter((it) => fold(it).includes(fold(value)))
	);
</script>

<!--
  Free-text autocomplete: the bound `value` IS the typed text, never constrained
  to `items` (which are mere suggestions, like a native datalist).

  bits-ui surfaces the displayed input text via the Root `inputValue` prop (its
  internal `oninput` writes the typed text there). We mirror that into our bound
  `value` two ways:
    - `oninput` on the input captures arbitrary typed text;
    - `onValueChange` on Root fires when a suggestion is picked.
  Passing `inputValue={value}` seeds the displayed text from an initial value.
-->
<Combobox.Root
	type="single"
	bind:open
	{disabled}
	inputValue={value}
	onValueChange={(v) => {
		// Ignore deselect: bits-ui allowDeselect fires onValueChange('') when the
		// selected item is re-clicked. Keep the last committed text instead of blanking.
		if (v) value = v;
	}}
>
	<div class="relative">
		<Combobox.Input
			{id}
			{name}
			{placeholder}
			{required}
			{disabled}
			oninput={(e) => {
				value = e.currentTarget.value;
				open = true;
			}}
			class="w-full rounded-lg border border-input bg-transparent px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
		/>
		{#if filtered.length > 0}
			<Combobox.Portal>
				<Combobox.Content
					class="z-50 max-h-60 overflow-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
				>
					<Combobox.Viewport>
						{#each filtered as item (item)}
							<Combobox.Item
								value={item}
								label={item}
								class="cursor-pointer rounded px-2 py-1.5 text-sm data-highlighted:bg-accent data-highlighted:text-accent-foreground"
							>
								{item}
							</Combobox.Item>
						{/each}
					</Combobox.Viewport>
				</Combobox.Content>
			</Combobox.Portal>
		{/if}
	</div>
</Combobox.Root>
