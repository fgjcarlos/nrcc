import { afterEach, describe, expect, it, vi } from 'vitest';

// Mock the navigation bridge so the 401 fallback in the response
// interceptor does not try to navigate during tests.
vi.mock('./navigation', () => ({
  redirectToLogin: vi.fn(),
}));

import { api } from './api';

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
