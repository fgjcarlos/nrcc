import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { afterEach, describe, expect, it } from 'vitest';
import { server } from '@/test/msw/server';
import { authService } from '../services/authService';
import { api } from '@/shared/lib/api';
import { useAuth } from './useAuth';

const ok = <T>(data: T) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

describe('useAuth bootstrap', () => {
  afterEach(() => {
    cleanup();
    authService.setToken(null);
  });

  it('TestUseAuth_BootWithRefreshCookie', async () => {
    authService.setToken(null);

    const { result } = renderHook(() => useAuth());

    await waitFor(() => {
      expect(result.current).toMatchObject({
        isAuthenticated: true,
        isInitialized: true,
        isLoading: false,
        user: {
          id: 'user-admin',
          username: 'admin',
          role: 'admin',
        },
      });
    });
    expect(authService.getToken()).toBe('nrcc-test-token');
  });

  it('shares one refresh across concurrent bootstraps and a 401 API caller', async () => {
    authService.setToken(null);
    let resolveRefresh!: () => void;
    let signalRefreshStarted!: () => void;
    const refreshStarted = new Promise<void>((resolve) => {
      signalRefreshStarted = resolve;
    });
    const refreshResponse = new Promise<void>((resolve) => {
      resolveRefresh = resolve;
    });
    let refreshCount = 0;

    server.use(
      http.post('/api/auth/refresh', async () => {
        refreshCount += 1;
        signalRefreshStarted();
        await refreshResponse;
        return ok({ token: 'shared-refresh-token' });
      }),
      http.get('/api/auth/me', ({ request }) =>
        request.headers.get('authorization') === 'Bearer shared-refresh-token'
          ? ok({ id: 'user-admin', username: 'admin', role: 'admin' })
          : HttpResponse.json({ success: false }, { status: 401 }),
      ),
    );

    const first = renderHook(() => useAuth());
    const second = renderHook(() => useAuth());
    await refreshStarted;
    authService.setToken('stale-access-token');
    const unauthorizedCall = api.get('/auth/me');

    // The 401 caller and both hooks must join the refresh already in flight.
    await waitFor(() => expect(refreshCount).toBe(1));
    resolveRefresh();
    await Promise.all([
      unauthorizedCall,
      waitFor(() => expect(first.result.current.isAuthenticated).toBe(true)),
      waitFor(() => expect(second.result.current.isAuthenticated).toBe(true)),
    ]);
    expect(refreshCount).toBe(1);
    expect(authService.getToken()).toBe('shared-refresh-token');
  });

  it('releases failed refresh coordination for retry and allows sequential refreshes', async () => {
    authService.setToken(null);
    let refreshCount = 0;
    server.use(
      http.post('/api/auth/refresh', () => {
        refreshCount += 1;
        if (refreshCount === 1) return HttpResponse.json({ success: false }, { status: 401 });
        return ok({ token: `sequential-token-${refreshCount}` });
      }),
      http.get('/api/auth/me', () => ok({ id: 'user-admin', username: 'admin', role: 'admin' })),
    );

    const { result } = renderHook(() => useAuth());
    await waitFor(() => expect(result.current.isInitialized).toBe(true));
    expect(result.current.isAuthenticated).toBe(false);
    expect(authService.getToken()).toBeNull();

    await act(() => result.current.checkAuth());
    expect(result.current.isAuthenticated).toBe(true);
    expect(authService.getToken()).toBe('sequential-token-2');

    authService.setToken(null);
    await act(() => result.current.checkAuth());
    expect(authService.getToken()).toBe('sequential-token-3');
    expect(refreshCount).toBe(3);
  });
});
