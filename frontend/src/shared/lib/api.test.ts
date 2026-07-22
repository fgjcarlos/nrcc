import { afterEach, describe, expect, it, vi } from 'vitest';

// Mock the navigation bridge so the 401 fallback in the response
// interceptor does not try to navigate during tests.
vi.mock('./navigation', () => ({
  redirectToLogin: vi.fn(),
}));

import { api, armAuthBootstrap, releaseAuthBootstrap } from './api';

describe('axios client (issue #362 — rehydrate on F5)', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('sends requests with withCredentials: true so the httpOnly refresh cookie travels on every call', () => {
    // axios.create stores the merged config on the instance. The
    // `defaults` is what every request picks up.
    const cfg = (api.defaults as { withCredentials?: boolean }) ?? {};
    expect(cfg.withCredentials).toBe(true);
  });

  it('forwards the httpOnly refresh cookie on the /auth/refresh call (so useAuth.checkAuth can rehydrate the session on F5)', async () => {
    // The critical client contract for issue #362: when useAuth calls
    // api.post('/auth/refresh', null), the request must carry
    // withCredentials: true. Without it, the httpOnly nrcc_refresh
    // cookie never reaches the server, the refresh returns 401, and
    // ProtectedRoute redirects to /login.
    const seenConfigs: Array<unknown> = [];
    const apiPost = vi
      .spyOn(api, 'post')
      .mockImplementation((_url: string, _body?: unknown, config?: unknown) => {
        seenConfigs.push(config);
        return Promise.resolve({ data: { data: { token: 'refreshed-token' } } });
      });

    await api.post('/auth/refresh', null);

    expect(apiPost).toHaveBeenCalledTimes(1);
    // axios normalises the per-call config; we just need to confirm
    // withCredentials ended up true on the request.
    const callConfig = (seenConfigs[0] ?? {}) as { withCredentials?: boolean };
    const withCredsFromConfig =
      callConfig.withCredentials ?? (api.defaults as { withCredentials?: boolean }).withCredentials;
    expect(withCredsFromConfig).toBe(true);

    apiPost.mockRestore();
  });
});

describe('auth bootstrap gate (issue #517 — race between useAuth and TanStack queries)', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('blocks non-auth requests until the bootstrap gate is released', async () => {
    const seen: string[] = [];

    // Arm the gate BEFORE the request fires. The first request
    // should wait; after we release, the request should pass.
    armAuthBootstrap();

    // Spy on the underlying adapter so we can see when the request
    // actually goes out.
    const adapter = vi.fn(async (config: { url?: string }) => {
      seen.push(config.url ?? '');
      return { data: { success: true }, status: 200, statusText: 'OK', headers: {}, config };
    });
    (api.defaults as { adapter?: unknown }).adapter = adapter;

    const blocked = api.get('/updates/status');
    // Yield to the microtask queue so the request interceptor runs
    // but the request does NOT go out yet.
    await Promise.resolve();
    await Promise.resolve();
    expect(seen).toEqual([]);

    releaseAuthBootstrap();
    await blocked;
    expect(seen).toEqual(['/updates/status']);
  });

  it('does not gate auth-bootstrap requests themselves (login/setup/refresh/me/status)', async () => {
    const seen: string[] = [];
    armAuthBootstrap();
    const adapter = vi.fn(async (config: { url?: string }) => {
      seen.push(config.url ?? '');
      return { data: { success: true }, status: 200, statusText: 'OK', headers: {}, config };
    });
    (api.defaults as { adapter?: unknown }).adapter = adapter;

    // All five auth-bootstrap paths must NOT be gated, even if the
    // gate is still pending. They are the bootstrap itself.
    await Promise.all([
      api.post('/auth/refresh', null),
      api.post('/auth/login', {}),
      api.post('/auth/setup', {}),
      api.get('/auth/me'),
      api.get('/auth/status'),
    ]);

    expect(seen).toEqual([
      '/auth/refresh',
      '/auth/login',
      '/auth/setup',
      '/auth/me',
      '/auth/status',
    ]);

    releaseAuthBootstrap();
  });

  it('stays armed while a refcount > 1 (second useAuth mount does not deadlock its own requests)', async () => {
    const seen: string[] = [];
    const adapter = vi.fn(async (config: { url?: string }) => {
      seen.push(config.url ?? '');
      return { data: { success: true }, status: 200, statusText: 'OK', headers: {}, config };
    });
    (api.defaults as { adapter?: unknown }).adapter = adapter;

    // armAuthBootstrap is single-shot: a second arm in the same
    // page load does not create a fresh pending gate. The first
    // arm is the only one that arms, the first release is the
    // only one that releases.
    armAuthBootstrap();
    armAuthBootstrap();

    const blocked = api.get('/updates/status');
    await Promise.resolve();
    await Promise.resolve();
    expect(seen).toEqual([]);

    // The first release unlocks the gate. A second release is a
    // no-op (the gate was already resolved).
    releaseAuthBootstrap();
    releaseAuthBootstrap();
    await blocked;
    expect(seen).toEqual(['/updates/status']);
  });

  it('does not deadlock when a second useAuth mount happens after the first has already released', async () => {
    const seen: string[] = [];
    const adapter = vi.fn(async (config: { url?: string }) => {
      seen.push(config.url ?? '');
      return { data: { success: true }, status: 200, statusText: 'OK', headers: {}, config };
    });
    (api.defaults as { adapter?: unknown }).adapter = adapter;

    // First useAuth: arm, release (gate resolves).
    armAuthBootstrap();
    releaseAuthBootstrap();

    // Second useAuth (e.g. on a route mounted after login). It
    // tries to arm again — the gate is already past its arm window,
    // so a fresh arm is a no-op. The gate stays resolved.
    armAuthBootstrap();

    // This request must pass through immediately, not deadlock.
    const r = await api.get('/updates/status');
    expect(seen).toEqual(['/updates/status']);
    expect(r.status).toBe(200);
  });
});
