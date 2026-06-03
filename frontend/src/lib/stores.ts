import { persisted } from 'svelte-persisted-store';

// Keys preserved verbatim from the legacy app for user-data continuity (Appendix D).
export const userName = persisted<string>('user_name', '');
export const userPhone = persisted<string>('user_phone', '');
export const lastOrigin = persisted<string>('last_origin', '');
export const lastDestination = persisted<string>('last_destination', '');
