// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { writable } from 'svelte/store';
export const a2hsModalOpen = writable(false);
export const openA2HS = () => a2hsModalOpen.set(true);
