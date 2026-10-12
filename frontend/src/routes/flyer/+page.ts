// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

// Like /about, this page is prerendered to static HTML (SSR on so the
// prerenderer can render it at build time). Host-dependent values (URL, QR)
// are filled in client-side on mount.
export const prerender = true;
export const ssr = true;
