import { writable } from 'svelte/store';
export const a2hsModalOpen = writable(false);
export const openA2HS = () => a2hsModalOpen.set(true);
