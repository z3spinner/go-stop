import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { api, ApiError } from './api';

/**
 * These tests exercise the real facade + generated client + mutator chain,
 * stubbing only the global `fetch`. They assert the behavioural contract from
 * Appendix A: error envelope -> ApiError, 204 -> null, X-Phone on owner reads,
 * `phone` in the body on mutations, and empty query params omitted. The exact
 * URLs/headers asserted are what the generated client + mutator actually emit.
 */

/** Build a Response-like stub good enough for the mutator (status, text, ok). */
function mockResponse(status: number, body: unknown): Response {
	const text = body === undefined ? '' : typeof body === 'string' ? body : JSON.stringify(body);
	return {
		status,
		ok: status >= 200 && status < 300,
		statusText: 'STATUS',
		text: () => Promise.resolve(text)
	} as unknown as Response;
}

const fetchMock = vi.fn();

beforeEach(() => {
	fetchMock.mockReset();
	vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

/** Pull (url, init) from the most recent fetch call. */
function lastCall(): { url: string; init: RequestInit } {
	const [url, init] = fetchMock.mock.calls.at(-1) as [string, RequestInit];
	return { url, init: init ?? {} };
}

describe('api facade — happy path / JSON parsing', () => {
	it('config.get() prepends /api and resolves parsed JSON', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, { siteName: 'Go-Stop', returnDelayHours: 2 }));
		const cfg = await api.config.get();
		expect(cfg).toEqual({ siteName: 'Go-Stop', returnDelayHours: 2 });
		expect(lastCall().url).toBe('/api/config');
	});
});

describe('api facade — error envelope', () => {
	it('rejects with ApiError carrying status + {error} message on 403', async () => {
		fetchMock.mockResolvedValue(mockResponse(403, { error: 'not the owner' }));
		await expect(api.rides.del('r1', '0600000000')).rejects.toMatchObject({
			name: 'ApiError',
			status: 403,
			message: 'not the owner'
		});
		await expect(api.rides.del('r1', '0600000000')).rejects.toBeInstanceOf(ApiError);
	});
});

describe('api facade — 204 / empty', () => {
	it('feedback() resolves null on 204 No Content', async () => {
		fetchMock.mockResolvedValue(mockResponse(204, undefined));
		const res = await api.rides.feedback('r1', '0600000000', true);
		expect(res).toBeNull();
	});
});

describe('api facade — owner-scoped read sends X-Phone', () => {
	it('requests.list(phone) sends the X-Phone header and no body', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, []));
		await api.requests.list('0612345678');
		const { url, init } = lastCall();
		expect(url).toBe('/api/requests');
		expect(init.method).toBe('GET');
		const headers = init.headers as Record<string, string>;
		expect(headers['X-Phone']).toBe('0612345678');
		expect(init.body).toBeUndefined();
	});

	it('interests.getContact(id, phone) sends X-Phone on the contact read', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, { phone: 'x', name: 'A', role: 'driver' }));
		await api.interests.getContact('i9', '0699999999');
		const { url, init } = lastCall();
		expect(url).toBe('/api/interests/i9/contact');
		expect((init.headers as Record<string, string>)['X-Phone']).toBe('0699999999');
	});
});

describe('api facade — mutation sends phone in JSON body', () => {
	it('rides.del(id, phone) puts phone in the DELETE body', async () => {
		fetchMock.mockResolvedValue(mockResponse(204, undefined));
		await api.rides.del('r7', '0600000001');
		const { url, init } = lastCall();
		expect(url).toBe('/api/rides/r7');
		expect(init.method).toBe('DELETE');
		expect(JSON.parse(init.body as string)).toEqual({ phone: '0600000001' });
	});

	it('interests.express(rideId, phone, name) puts phone+name in the POST body', async () => {
		fetchMock.mockResolvedValue(mockResponse(201, { id: 'i1', status: 'pending' }));
		await api.interests.express('r3', '0600000002', 'Alice');
		const { url, init } = lastCall();
		expect(url).toBe('/api/rides/r3/interest');
		expect(init.method).toBe('POST');
		expect(JSON.parse(init.body as string)).toEqual({ phone: '0600000002', name: 'Alice' });
	});

	it('requests.ping(id, rideId, phone) sends ride_id in body AND X-Phone header', async () => {
		fetchMock.mockResolvedValue(mockResponse(204, undefined));
		await api.requests.ping('q1', 'r5', '0600000003');
		const { url, init } = lastCall();
		expect(url).toBe('/api/requests/q1/ping');
		expect(init.method).toBe('POST');
		expect(JSON.parse(init.body as string)).toEqual({ ride_id: 'r5' });
		expect((init.headers as Record<string, string>)['X-Phone']).toBe('0600000003');
	});
});

describe('api facade — query building', () => {
	it('rides.list({}) omits empty query params (bare /api/rides)', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, []));
		await api.rides.list();
		expect(lastCall().url).toBe('/api/rides');
	});

	it('rides.list(params) includes only the provided params', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, []));
		await api.rides.list({ origin: 'Lyon', destination: 'Paris' });
		const { url } = lastCall();
		expect(url).toContain('/api/rides?');
		expect(url).toContain('origin=Lyon');
		expect(url).toContain('destination=Paris');
		expect(url).not.toContain('search_date');
	});

	it('rides.list(params, phone) for my-rides adds X-Phone', async () => {
		fetchMock.mockResolvedValue(mockResponse(200, []));
		await api.rides.list({}, '0600000004');
		const { init } = lastCall();
		expect((init.headers as Record<string, string>)['X-Phone']).toBe('0600000004');
	});
});
