import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useEnvironment } from './useEnvironment';
import { bootstrapService } from '@/features/bootstrap/services';

vi.mock('@/features/bootstrap/services', () => ({
  bootstrapService: {
    getStatus: vi.fn(),
  },
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useEnvironment hook', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('unwraps the host status from the API response envelope', async () => {
    const hostStatus = {
      nodeRedInstalled: true,
      nodeRedRunning: false,
      nodeVersion: 'v22.0.0',
    };
    vi.mocked(bootstrapService.getStatus).mockResolvedValue({
      data: { data: hostStatus },
    } as unknown as Awaited<ReturnType<typeof bootstrapService.getStatus>>);

    const { result } = renderHook(() => useEnvironment(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(hostStatus);
    expect(bootstrapService.getStatus).toHaveBeenCalledTimes(1);
  });

  it('surfaces an error when the bootstrap service fails', async () => {
    vi.mocked(bootstrapService.getStatus).mockRejectedValue(new Error('network down'));

    const { result } = renderHook(() => useEnvironment(), { wrapper: createWrapper() });

    // The hook sets retry: 2 (overriding the client default), so it retries with
    // backoff before surfacing the error — allow time for those retries.
    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 8000 });
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
