// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { writable } from 'svelte/store';
import type { PushState } from './pwa';

// null = closed; otherwise the state the modal should present.
export const notifModalState = writable<PushState | null>(null);
export function openNotifModal(state: PushState) {
	notifModalState.set(state);
}
