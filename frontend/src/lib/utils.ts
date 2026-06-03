import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { getLocale } from '$lib/paraglide/runtime';
import { m } from '$lib/paraglide/messages';
import type { Flexibility } from './types';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, "child"> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, "children"> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };

const BCP47: Record<string, string> = {
	fr: 'fr-FR', en: 'en-GB', es: 'es-ES', it: 'it-IT', de: 'de-DE', nl: 'nl-NL'
};

export function localeToBCP47(locale: string = getLocale()): string {
	return BCP47[locale] ?? 'en-GB';
}

export function normalizePhone(phone: string): string {
	return phone.trim().replace(/[\s.\-()]/g, '');
}

/** Compact flexibility tag, e.g. "Exact" / "±30 min" / "±60 min". */
export function flexLabel(flex: Flexibility | number): string {
	if (flex === 30) return m.flexLabel30();
	if (flex === 60) return m.flexLabel60();
	if (flex === 0) return m.flexLabelExact();
	return `${flex} min`;
}

function pad(n: number): string {
	return String(n).padStart(2, '0');
}

/** Local now + 1h, rounded up to the next 5 minutes, as a `datetime-local` value. */
export function defaultDeparture(): string {
	const d = new Date(Date.now() + 60 * 60 * 1000);
	const rem = d.getMinutes() % 5;
	if (rem !== 0) d.setMinutes(d.getMinutes() + (5 - rem), 0, 0);
	d.setSeconds(0, 0);
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** "Wed 4 Jun at 14:30" in the active locale. */
export function formatTime(iso: string): string {
	const loc = localeToBCP47();
	const d = new Date(iso);
	const date = d.toLocaleDateString(loc, { weekday: 'short', day: 'numeric', month: 'short' });
	const time = d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' });
	return `${date} ${m.at()} ${time}`;
}

/** "Wed 4 Jun" in the active locale. */
export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString(localeToBCP47(), {
		weekday: 'short', day: 'numeric', month: 'short'
	});
}
