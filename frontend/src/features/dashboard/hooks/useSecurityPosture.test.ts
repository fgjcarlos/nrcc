import React from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useSecurityPosture } from './useSecurityPosture';
import { dashboardService } from '../services';

const authState: { user: { role: 'admin' | 'viewer' } | undefined } = { user: { role: 'admin' } };

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: () => authState,
}));

function createWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
}

describe('useSecurityPosture', () => {
  beforeEach(() => {
    authState.user = { role: 'admin' };
    vi.restoreAllMocks();
  });

  it('fetches and returns the canonical posture fields for an admin', async () => {
    const { result } = renderHook(() => useSecurityPosture(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.data).toEqual({
      encryptionKeyConfigured: true,
      backupDownloadAdminOnly: true,
      activeRefreshSessions: 0,
      mfa: { enrolledAdmins: 2, totalAdmins: 2 },
    }));
  });

  it('does not request posture for a viewer', async () => {
    authState.user = { role: 'viewer' };
    const request = vi.spyOn(dashboardService, 'getSecurityPosture');

    const { result } = renderHook(() => useSecurityPosture(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.fetchStatus).toBe('idle'));
    expect(request).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
  });

  it('keeps error and absent results unavailable instead of fabricating a healthy posture', async () => {
    vi.spyOn(dashboardService, 'getSecurityPosture').mockRejectedValueOnce(new Error('unavailable'));
    const { result } = renderHook(() => useSecurityPosture(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
  });
});
