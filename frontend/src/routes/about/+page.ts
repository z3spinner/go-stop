// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// The rest of the app is a client-only SPA (see ../+layout.ts). This page is the
// exception: prerender it to static HTML so the About content is crawlable and
// indexable without depending on Googlebot executing JS. SSR must be enabled for
// the route so the prerenderer can render it at build time. Live users still get
// all languages via client hydration + reactive messages.
export const prerender = true;
export const ssr = true;
