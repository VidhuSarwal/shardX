import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, apiRequest, clearAuthToken, setAuthToken } from '@/lib/api';

// jsdom-free: stub the browser globals the client touches.
const store = new Map<string, string>();
const localStorageMock = {
  getItem: (k: string) => store.get(k) ?? null,
  setItem: (k: string, v: string) => void store.set(k, v),
  removeItem: (k: string) => void store.delete(k),
};

beforeEach(() => {
  store.clear();
  vi.stubGlobal('localStorage', localStorageMock);
  vi.stubGlobal('window', { dispatchEvent: () => true, location: { href: '' } });
});
afterEach(() => vi.unstubAllGlobals());

describe('apiRequest', () => {
  it('sends the bearer token and JSON content type', async () => {
    setAuthToken('tok', 'me@x.io');
    const fetchMock = vi.fn(async () => new Response('{"ok":true}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    await apiRequest('/api/x', { method: 'POST', body: '{}' });

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe('http://localhost:5555/api/x');
    const h = init.headers as Headers;
    expect(h.get('Authorization')).toBe('Bearer tok');
    expect(h.get('Content-Type')).toBe('application/json');
  });

  it("surfaces the backend's plain-text error with its status", async () => {
    vi.stubGlobal('fetch', async () => new Response('upload incomplete: 1/2 bytes\n', { status: 400 }));
    const err = await apiRequest('/api/x').catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(400);
    expect((err as ApiError).message).toBe('upload incomplete: 1/2 bytes');
  });

  it('clears the session and redirects on 401 unless skipAuthRedirect', async () => {
    setAuthToken('tok');
    vi.stubGlobal('fetch', async () => new Response('unauthorized', { status: 401 }));

    await expect(apiRequest('/api/x', { skipAuthRedirect: true })).rejects.toThrow('unauthorized');
    expect(store.get('auth_token')).toBe('tok');

    await expect(apiRequest('/api/x')).rejects.toThrow('unauthorized');
    expect(store.get('auth_token')).toBeUndefined();
    expect((globalThis as { window: { location: { href: string } } }).window.location.href).toBe('/login');
  });

  it('clearAuthToken also forgets the remembered email', () => {
    setAuthToken('tok', 'me@x.io');
    clearAuthToken();
    expect(store.get('auth_email')).toBeUndefined();
  });
});
