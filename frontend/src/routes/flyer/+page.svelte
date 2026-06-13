<!--
  SPDX-FileCopyrightText: 2026 Zeno Kerr
  SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { m } from '$lib/paraglide/messages';
	import { config } from '$lib/config';
	import { getLocale, locales, baseLocale, type Locale } from '$lib/locale';

	// `selected` drives only the flyer's rendered copy (via Paraglide's per-call
	// { locale } option) — it never touches the app's global locale or reloads.
	let selected = $state<Locale>(baseLocale);
	let origin = $state('');
	let qrSvg = $state('');

	let host = $derived(origin.replace(/^https?:\/\//, ''));

	// The accolade adapts per community: strip the "Go Stop" brand prefix and any
	// trailing punctuation from the site name to get the place (e.g.
	// "Go Stop Saillans!" → "Saillans"). Empty (e.g. the bare "Go-Stop" default)
	// hides the accolade.
	let place = $derived(
		$config.siteName.replace(/^go[\s-]*stop[\s-]*/i, '').replace(/[!.\s]+$/, '').trim()
	);

	const localeNames: Record<Locale, string> = {
		fr: 'FR', en: 'EN', es: 'ES', it: 'IT', de: 'DE', nl: 'NL', el: 'EL'
	};

	// Shrink the URL's font so the full host always fits on one line — live
	// domains are longer than the dev host and would otherwise overflow.
	const URL_BASE_PX = 40;
	let urlEl: HTMLSpanElement | undefined = $state();
	function fitUrl() {
		if (!urlEl) return;
		urlEl.style.fontSize = URL_BASE_PX + 'px';
		const ratio = urlEl.scrollWidth / urlEl.clientWidth;
		if (ratio > 1) urlEl.style.fontSize = Math.max(13, (URL_BASE_PX / ratio) * 0.96) + 'px';
	}

	// Re-fit whenever the host resolves, and on resize. Run once more after the
	// monospace font loads so the measurement reflects the real glyph widths.
	$effect(() => {
		host; // track
		fitUrl();
	});
	$effect(() => {
		window.addEventListener('resize', fitUrl);
		return () => window.removeEventListener('resize', fitUrl);
	});

	onMount(async () => {
		selected = getLocale();
		origin = window.location.origin;
		try {
			qrSvg = await QRCode.toString(origin, {
				type: 'svg',
				margin: 0,
				errorCorrectionLevel: 'M'
			});
		} catch {
			/* leave QR empty if generation fails */
		}
		await document.fonts?.ready;
		fitUrl();
	});
</script>

<svelte:head>
	<title>{m.flyerMetaTitle({}, { locale: selected })} · {$config.siteName}</title>
	<meta
		name="description"
		content={m.flyerMetaDescription({ siteName: $config.siteName }, { locale: selected })}
	/>
	<link rel="canonical" href="/flyer" />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,600;0,9..144,900;1,9..144,600;1,9..144,800&family=Bricolage+Grotesque:wght@500;600;700;800&family=Space+Mono:wght@700&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<div class="flyer-controls no-print">
	<div class="lang-row">
		{#each locales as l (l)}
			<button
				type="button"
				class="lang-btn"
				class:active={l === selected}
				aria-pressed={l === selected}
				onclick={() => (selected = l)}>{localeNames[l]}</button
			>
		{/each}
	</div>
	<button type="button" class="print-btn" onclick={() => window.print()}>
		{m.flyerPrint({}, { locale: selected })}
	</button>
</div>

<div class="flyer-stage">
	<div class="poster">
		<span class="pin l"></span><span class="pin r"></span>

		<div class="brand">
			<svg class="mark" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
				<defs>
					<clipPath id="flyerClip"><rect width="512" height="512" rx="115" /></clipPath>
					<linearGradient id="flyerGl" x1="0" y1="0" x2="0.8" y2="1">
						<stop offset="0" stop-color="#4CD964" /><stop offset="1" stop-color="#28A836" />
					</linearGradient>
					<linearGradient id="flyerGr" x1="0.2" y1="0" x2="1" y2="1">
						<stop offset="0" stop-color="#FF4538" /><stop offset="1" stop-color="#C81E1E" />
					</linearGradient>
				</defs>
				<g clip-path="url(#flyerClip)">
					<rect width="256" height="512" fill="url(#flyerGl)" />
					<rect x="256" width="256" height="512" fill="url(#flyerGr)" />
				</g>
				<g fill="none" stroke="#fff" stroke-width="22" stroke-linecap="round" stroke-linejoin="round">
					<path d="M 196,175 A 106,106 0 1 0 234,256 L 278,256" /><circle cx="384" cy="256" r="106" />
				</g>
				<polygon points="90,196 90,316 198,256" fill="#fff" />
				<rect x="330" y="202" width="108" height="108" rx="20" fill="#fff" />
			</svg>
			<span class="wordmark">{$config.siteName}</span>
		</div>

		<h1 class="hero">
			{m.flyerHeadlineLine1({}, { locale: selected })}<em>{m.flyerHeadlineLine2({}, { locale: selected })}</em>
		</h1>
		<p class="sub">{m.tagline({}, { locale: selected })}</p>

		<div class="route">
			<div class="stop">
				<div class="num">1</div>
				<b>{m.flyerStep1Title({}, { locale: selected })}</b>
				<span>{m.flyerStep1Desc({}, { locale: selected })}</span>
			</div>
			<div class="stop">
				<div class="num">2</div>
				<b>{m.flyerStep2Title({}, { locale: selected })}</b>
				<span>{m.flyerStep2Desc({}, { locale: selected })}</span>
			</div>
			<div class="stop">
				<div class="num">3</div>
				<b>{m.flyerStep3Title({}, { locale: selected })}</b>
				<span>{m.flyerStep3Desc({}, { locale: selected })}</span>
			</div>
		</div>

		<div class="dest">
			<div class="main">
				<span class="label">{m.flyerCtaLabel({}, { locale: selected })}</span>
				<span class="url" bind:this={urlEl}>{host}</span>
			</div>
			<div class="qr">
				<div class="qrimg">{@html qrSvg}</div>
				<span>{m.flyerScan({}, { locale: selected })}</span>
			</div>
		</div>

		{#if place}
			<p class="made-in">♥ {m.flyerMadeIn({ place }, { locale: selected })}</p>
		{/if}

		<div class="foot">
			<div class="pillrow">
				<span class="pill">{m.flyerBadgeNoAds({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeNoTracking({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeNoCookies({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeNoAccount({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeFree({}, { locale: selected })}</span>
				<span class="pill">{m.flyerBadgeOrganic({}, { locale: selected })}</span>
			</div>
			<span class="tear">♺ {m.flyerPinMe({}, { locale: selected })}</span>
		</div>
	</div>
</div>

<style>
	/* ---- screen controls (never printed) ---- */
	.flyer-controls {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 14px;
		flex-wrap: wrap;
		padding: 16px;
	}
	.lang-row { display: flex; gap: 6px; flex-wrap: wrap; }
	.lang-btn {
		font: 700 12px/1 system-ui, sans-serif;
		padding: 6px 10px;
		border: 1px solid #cfd8cf;
		border-radius: 8px;
		background: #fff;
		color: #2b332b;
		cursor: pointer;
	}
	.lang-btn.active { background: #1f9d3a; border-color: #1f9d3a; color: #fff; }
	.print-btn {
		font: 700 13px/1 system-ui, sans-serif;
		padding: 9px 18px;
		border: none;
		border-radius: 999px;
		background: #1f9d3a;
		color: #fff;
		cursor: pointer;
	}

	.flyer-stage { display: flex; justify-content: center; padding: 0 16px 32px; }

	/* ---- the poster (A4 portrait, ~720x1018 px ≈ 190x269 mm) ---- */
	.poster {
		--green: #1f9d3a; --green-deep: #15772b; --ink: #1f201a; --paper: #fbf6ea;
		position: relative;
		width: 720px;
		min-height: 1018px;
		background: var(--paper);
		color: var(--ink);
		border-radius: 6px;
		overflow: hidden;
		padding: 60px 64px 46px;
		box-shadow: 0 18px 50px rgba(40, 30, 10, 0.28);
		display: flex;
		flex-direction: column;
		font-family: 'Bricolage Grotesque', sans-serif;
		print-color-adjust: exact;
		-webkit-print-color-adjust: exact;
	}
	.poster::before {
		content: '';
		position: absolute;
		inset: 0;
		pointer-events: none;
		background:
			radial-gradient(120% 80% at 50% -10%, #ffffff 0%, transparent 55%),
			radial-gradient(140% 100% at 50% 120%, rgba(180, 150, 90, 0.18) 0%, transparent 60%);
	}
	.poster::after {
		content: '';
		position: absolute;
		inset: 0;
		pointer-events: none;
		opacity: 0.45;
		mix-blend-mode: multiply;
		background-image: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='120' height='120'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/><feColorMatrix type='saturate' values='0'/></filter><rect width='100%25' height='100%25' filter='url(%23n)' opacity='0.5'/></svg>");
	}
	.poster > * { position: relative; z-index: 1; }

	.pin {
		position: absolute;
		width: 19px;
		height: 19px;
		border-radius: 50%;
		background: radial-gradient(circle at 35% 30%, #fff 0 18%, #cf3b2e 22% 60%, #8e231a 100%);
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.35);
		z-index: 2;
	}
	.pin.l { top: 22px; left: 22px; }
	.pin.r { top: 22px; right: 22px; }

	.brand { display: flex; align-items: center; gap: 18px; }
	.mark { width: 64px; height: 64px; border-radius: 16px; box-shadow: 0 3px 10px rgba(0, 0, 0, 0.18); flex: none; }
	.wordmark { font-weight: 800; font-size: 36px; letter-spacing: -0.02em; }

	.hero {
		font-family: 'Fraunces', serif;
		font-weight: 900;
		font-size: 66px;
		line-height: 0.96;
		letter-spacing: -0.02em;
		margin: 42px 0 0;
	}
	.hero em { font-style: italic; font-weight: 800; color: var(--green); display: block; }

	.sub {
		font-family: 'Fraunces', serif;
		font-style: italic;
		font-weight: 600;
		font-size: 26px;
		line-height: 1.35;
		color: #5b5a4e;
		margin: 24px auto 0;
		max-width: 540px;
		text-align: center;
	}

	.route { display: flex; justify-content: space-between; position: relative; margin: 44px 6px 0; }
	.route::before {
		content: '';
		position: absolute;
		top: 27px;
		left: 8%;
		right: 8%;
		border-top: 4px dashed #c9bfa3;
		z-index: 0;
	}
	.stop {
		position: relative;
		z-index: 1;
		width: 31%;
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
	}
	.num {
		width: 55px;
		height: 55px;
		border-radius: 50%;
		background: var(--green);
		color: #fff;
		font-weight: 800;
		font-size: 27px;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 5px 14px rgba(31, 157, 58, 0.4);
		border: 5px solid var(--paper);
	}
	.stop b { font-size: 21px; font-weight: 700; line-height: 1.15; }
	.stop span { font-size: 18px; color: #6c6a5d; line-height: 1.2; }

	.dest {
		margin: 46px 0 0;
		display: flex;
		align-items: stretch;
		border-radius: 28px;
		overflow: hidden;
		box-shadow: 0 12px 28px rgba(21, 119, 43, 0.34);
	}
	.dest .main {
		flex: 1;
		min-width: 0;
		background: linear-gradient(155deg, var(--green) 0%, var(--green-deep) 100%);
		color: #fff;
		padding: 32px 36px;
		display: flex;
		flex-direction: column;
		justify-content: center;
		gap: 8px;
	}
	.dest .label {
		font-size: 16px;
		font-weight: 700;
		letter-spacing: 0.24em;
		text-transform: uppercase;
		opacity: 0.85;
	}
	.dest .url {
		font-family: 'Space Mono', monospace;
		font-weight: 700;
		font-size: 40px;
		letter-spacing: -0.03em;
		line-height: 1.05;
		white-space: nowrap;
		/* font-size is auto-shrunk in JS (fitUrl) so the whole host always fits */
	}
	.dest .qr {
		background: #fff;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 25px;
		flex: none;
	}
	.dest .qrimg { width: 116px; height: 116px; }
	.dest .qrimg :global(svg) { width: 100%; height: 100%; display: block; }
	.dest .qr span {
		font-size: 14px;
		font-weight: 700;
		letter-spacing: 0.16em;
		color: #6c6a5d;
		text-transform: uppercase;
	}

	.made-in {
		margin: 16px auto 0;
		text-align: center;
		font-size: 13px;
		font-weight: 700;
		letter-spacing: 0.16em;
		text-transform: uppercase;
		color: var(--green-deep);
		opacity: 0.85;
	}

	/* margin-top:auto on the footer pools any slack BELOW the badges, keeping the
	   URL/destination box up as the focal point (the flagged polish item). */
	.foot { display: flex; align-items: center; gap: 12px; margin-top: auto; padding-top: 28px; }
	.pillrow { display: flex; gap: 10px; flex-wrap: wrap; }
	.pill {
		font-size: 17px;
		font-weight: 700;
		color: var(--green-deep);
		background: #e9f6ea;
		border-radius: 999px;
		padding: 7px 19px;
	}
	.tear { margin-left: auto; white-space: nowrap; font-size: 17px; font-style: italic; font-family: 'Fraunces', serif; color: #9a9685; }

	/* ---- print ---- */
	@media print {
		:global(.no-print) { display: none !important; }
		.flyer-stage { padding: 0; }
		.poster { box-shadow: none; border-radius: 0; }
	}
</style>
