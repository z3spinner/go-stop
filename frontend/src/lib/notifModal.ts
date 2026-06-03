import { writable } from 'svelte/store';
import type { PushState } from './pwa';

// null = closed; otherwise the state the modal should present.
export const notifModalState = writable<PushState | null>(null);
export function openNotifModal(state: PushState) {
	notifModalState.set(state);
}
