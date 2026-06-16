<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	let {
		value = $bindable(0),
		min = 0,
		max = 99
	}: { value?: number; min?: number; max?: number } = $props();

	function clamp(v: number): number {
		if (Number.isNaN(v)) return min;
		return Math.min(max, Math.max(min, Math.floor(v)));
	}
	function step(delta: number) {
		value = clamp(value + delta);
	}
	function onInput(e: Event) {
		value = clamp(Number((e.currentTarget as HTMLInputElement).value));
	}
</script>

<div class="number-stepper inline-flex items-stretch overflow-hidden rounded-lg border border-input">
	<button
		type="button"
		aria-label="decrease"
		class="px-3 text-lg leading-none text-foreground disabled:opacity-40"
		disabled={value <= min}
		onclick={() => step(-1)}>−</button>
	<input
		type="text"
		inputmode="numeric"
		class="w-12 border-x border-input bg-transparent text-center text-sm outline-none"
		value={value}
		onchange={onInput}
	/>
	<button
		type="button"
		aria-label="increase"
		class="px-3 text-lg leading-none text-foreground disabled:opacity-40"
		disabled={value >= max}
		onclick={() => step(1)}>+</button>
</div>
