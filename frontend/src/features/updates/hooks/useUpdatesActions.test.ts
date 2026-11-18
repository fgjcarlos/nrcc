import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { updateService } from '@/features/updates/services';
import { useUpdatesActions } from './useUpdatesActions';

const { toastError, toastSuccess } = vi.hoisted(() => ({ toastError: vi.fn(), toastSuccess: vi.fn() }));
vi.mock('sonner', () => ({ toast: { error: toastError, success: toastSuccess } }));
vi.mock('@/features/updates/services', () => ({
  updateService: { check: vi.fn(), applyUpdate: vi.fn() },
}));

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
  return renderHook(() => useUpdatesActions(), { wrapper });
}

const checked = { currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z' };

describe('useUpdatesActions check', () => {
  beforeEach(() => {
    toastError.mockReset();
    toastSuccess.mockReset();
  });

  it('reports success when the check returns fresh versions', async () => {
    vi.mocked(updateService.check).mockResolvedValue(checked);
    const { result } = setup();

    await act(() => result.current.checkMutation.mutateAsync());

    expect(toastSuccess).toHaveBeenCalledWith('Check completed');
    expect(toastError).not.toHaveBeenCalled();
  });

  it('reports the failure, not success, when the check carries an error', async () => {
    vi.mocked(updateService.check).mockResolvedValue({ ...checked, latestVersion: '', updateAvailable: false, error: 'Network error' });
    const { result } = setup();

    await act(() => result.current.checkMutation.mutateAsync());

    expect(toastError).toHaveBeenCalledWith('Error checking for updates');
    expect(toastSuccess).not.toHaveBeenCalled();
  });
});
