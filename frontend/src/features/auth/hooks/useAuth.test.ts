import { cleanup, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { authService } from '../services/authService';
import { useAuth } from './useAuth';

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
});
