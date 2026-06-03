import { persisted } from 'svelte-persisted-store';

// These hold plain strings and must be stored RAW (not JSON-encoded). The legacy
// app wrote them as raw values (e.g. user_name=Marie, not "Marie"), so existing
// users already have raw localStorage data — the default JSON serializer would
// throw on JSON.parse("Marie") and silently fall back to the empty default,
// losing the profile + last-search across every form. A pass-through serializer
// reads/writes the values verbatim, matching the legacy format (Appendix D).
//
// parse() also tolerates JSON-quoted values: a brief window of the refactored app
// shipped with the default serializer, so a few users may have a quoted "Marie"
// in storage. Unwrapping it (rather than showing literal quotes) heals that data
// on the next save, when stringify() rewrites it raw.
export const rawString = {
	parse: (value: string): string => {
		if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) {
			try {
				const unwrapped = JSON.parse(value);
				if (typeof unwrapped === 'string') return unwrapped;
			} catch {
				/* not valid JSON — fall through and treat as a raw value */
			}
		}
		return value;
	},
	stringify: (value: string): string => value
};

export const userName = persisted<string>('user_name', '', { serializer: rawString });
export const userPhone = persisted<string>('user_phone', '', { serializer: rawString });
export const lastOrigin = persisted<string>('last_origin', '', { serializer: rawString });
export const lastDestination = persisted<string>('last_destination', '', { serializer: rawString });
