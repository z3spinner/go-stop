<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type { Flexibility } from '$lib/types';

	let { value = $bindable(30) }: { value?: Flexibility } = $props();

	const options: { v: Flexibility; label: () => string }[] = [
		{ v: 0, label: m.flexExact },
		{ v: 30, label: m.flex30 },
		{ v: 60, label: m.flex60 }
	];
	function labelFor(v: Flexibility): string {
		return (options.find((o) => o.v === v) ?? options[1]).label();
	}

	// bits-ui Select works in strings; bridge to the numeric Flexibility.
	let strValue = $derived(String(value));
	function onValueChange(v: string) {
		value = Number(v) as Flexibility;
	}
</script>

<Select.Root type="single" value={strValue} {onValueChange}>
	<Select.Trigger class="w-full">{labelFor(value)}</Select.Trigger>
	<Select.Content>
		{#each options as o (o.v)}
			<Select.Item value={String(o.v)} label={o.label()}>{o.label()}</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
