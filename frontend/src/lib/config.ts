import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { api } from './api';
import type { Config } from './types';

export const config = writable<Config>({ siteName: 'Go-Stop', returnDelayHours: 2 });

export async function loadConfig(): Promise<void> {
	if (!browser) return;
	try {
		const c = await api.config.get();
		config.set(c);
		document.title = `${c.siteName}`;
	} catch {
		/* keep defaults */
	}
}
