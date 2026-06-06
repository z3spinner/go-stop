<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
</script>

<svelte:head>
	<title>{m.aboutTitle()} · {$config.siteName}</title>
	<meta name="description" content={m.aboutMetaDescription()} />
	<!-- Relative: this page is prerendered at build time with no known host, and
	     each deployed instance has its own domain. Google accepts relative canonicals. -->
	<link rel="canonical" href="/about" />
</svelte:head>

<h1 class="mb-3 text-xl font-semibold">{m.aboutTitle()}</h1>
<div class="modal-body">{@html m.aboutBody({ siteName: $config.siteName })}</div>
