import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/test/msw/server';
import { authUsersResponse, mockUser } from '@/test/msw/fixtures';

const ok = <T>(data: T) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

describe('authService in-memory token lifecycle', () => {
  beforeEach(() => {
    vi.resetModules();
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('TestAuthService_NoSessionStorage', async () => {
    sessionStorage.setItem('nrcc_access_token', 'persisted-token');
    const getItem = vi.spyOn(sessionStorage, 'getItem');
    const setItem = vi.spyOn(sessionStorage, 'setItem');
    const removeItem = vi.spyOn(sessionStorage, 'removeItem');
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }));

    const { authService } = await import('./authService');

    expect(authService.getToken()).toBeNull();
    await authService.login('admin', 'password123');
    expect(authService.getToken()).toBe('nrcc-test-token');
    await authService.logout();
    expect(authService.getToken()).toBeNull();
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/logout', {
      method: 'POST',
      headers: { Authorization: 'Bearer nrcc-test-token' },
      credentials: 'include',
    });
    expect(getItem).not.toHaveBeenCalled();
    expect(setItem).not.toHaveBeenCalled();
    expect(removeItem).not.toHaveBeenCalled();
  });

  it('TestAuthService_LogoutIsTerminal', async () => {
    vi.useFakeTimers();
    let resolveLogout!: (response: Response) => void;
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockReturnValue(
      new Promise<Response>((resolve) => {
        resolveLogout = resolve;
      }),
    );
    const { authService } = await import('./authService');
    authService.setToken('logout-token');

    const logoutPromise = authService.logout();
    expect(authService.getToken()).toBeNull();

    setTimeout(() => authService.setToken('racing-refresh-token'), 10);
    await vi.advanceTimersByTimeAsync(10);
    expect(authService.getToken()).toBe('racing-refresh-token');

    resolveLogout(new Response(null, { status: 204 }));
    await logoutPromise;

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/logout', {
      method: 'POST',
      headers: { Authorization: 'Bearer logout-token' },
      credentials: 'include',
    });
    expect(authService.getToken()).toBeNull();

    fetchMock.mockClear();
    await authService.logout();
    expect(fetchMock).not.toHaveBeenCalled();
    expect(authService.getToken()).toBeNull();

    fetchMock.mockRejectedValueOnce(new Error('network unavailable'));
    authService.setToken('failure-token');
    await expect(authService.logout()).rejects.toThrow('network unavailable');
    expect(authService.getToken()).toBeNull();
  });

  it('TestAuthService_RefreshRehydrates', async () => {
    server.use(
      http.post('/api/auth/refresh', () => ok({ token: 'refreshed-token' })),
      http.get('/api/auth/me', ({ request }) => {
        if (request.headers.get('authorization') === 'Bearer refreshed-token') {
          return ok(mockUser);
        }
        return HttpResponse.json({ success: false }, { status: 401 });
      }),
    );
    const { authService } = await import('./authService');
    const setToken = vi.spyOn(authService, 'setToken');

    const user = await authService.getMe();

    expect(setToken).toHaveBeenCalledWith('refreshed-token');
    expect(authService.getToken()).toBe('refreshed-token');
    expect(user).toEqual(mockUser);
  });
});

describe('authService setup token contract', () => {
  it('sends the setup token only in X-Setup-Reset-Token', async () => {
    let receivedToken: string | null = null
    let receivedUrl = ''
    let receivedBody: unknown
    server.use(
      http.post('/api/auth/setup', async ({ request }) => {
        receivedToken = request.headers.get('X-Setup-Reset-Token')
        receivedUrl = request.url
        receivedBody = await request.json()
        return ok({ token: 'access-token', user: mockUser })
      }),
    )

    const { authService } = await import('./authService')
    await authService.setup('admin', 'password123', 'host-secret')

    expect(receivedToken).toBe('host-secret')
    expect(receivedUrl).not.toContain('host-secret')
    expect(receivedBody).not.toHaveProperty('setupToken')
  })
})

describe('authService password change contract', () => {
  it('sends the current password with a self-service change and omits it for an admin reset', async () => {
    const bodies: unknown[] = []
    server.use(
      http.patch('/api/auth/users/:id/password', async ({ params, request }) => {
        bodies.push(await request.json())
        return ok(params.id === 'u1' ? { success: true, token: 'replacement' } : { success: true })
      }),
    )

    const { authService } = await import('./authService')
    await authService.changePassword('u1', 'new-password-123', 'old-password-123')
    expect(authService.getToken()).toBe('replacement')
    await authService.changePassword('u2', 'reset-password-123')
    expect(authService.getToken()).toBe('replacement')

    expect(bodies).toEqual([
      { password: 'new-password-123', currentPassword: 'old-password-123' },
      { password: 'reset-password-123' },
    ])
  })
})

describe('authService users contract', () => {
  it('decodes the backend users envelope', async () => {
    server.use(
      http.get('/api/auth/users', () => ok(authUsersResponse)),
    );

    const { authService } = await import('./authService');

    await expect(authService.getUsers()).resolves.toEqual([mockUser]);
  });
});
