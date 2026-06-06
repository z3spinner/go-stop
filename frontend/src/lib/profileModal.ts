// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { writable } from 'svelte/store';

// Holds the action to run once the profile is completed; null when closed.
export const profileModalState = writable<(() => void) | null>(null);

export function openProfileModal(onComplete: () => void) {
	profileModalState.set(onComplete);
}
