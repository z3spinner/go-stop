// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Orval `fetch` client mutator.
 *
 * orval's generated functions call `customFetch(url, init)` and expect the
 * parsed response body back (type T). This mutator centralises:
 *   - the `/api` base path (the swagger basePath; generated URLs are relative),
 *   - the `{error}` envelope -> ApiError(status, message) on non-2xx,
 *   - 204/empty bodies -> null.
 *
 * The facade (`$lib/api.ts`) re-exports ApiError and wraps the generated
 * functions into the Appendix A `api.<resource>.<verb>()` surface.
 */

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export async function customFetch<T>(url: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`/api${url}`, init);
	if (res.status === 204) return null as T;
	const text = await res.text();
	const data = text ? JSON.parse(text) : null;
	if (!res.ok) {
		const msg =
			data && typeof data === 'object' && 'error' in data
				? (data as { error: string }).error
				: res.statusText;
		throw new ApiError(res.status, msg);
	}
	return data as T;
}
