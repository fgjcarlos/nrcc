import { describe, expect, it, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useSystemHistory } from './useSystemHistory';
import * as historyServiceModule from '../services/historyService';
import type { MetricsSnapshot } from '../types/history';

vi.mock('../services/historyService', () => ({
  historyService: {
    getSystemHistory: vi.fn(),
  },
}));

const mockSnapshots: MetricsSnapshot[] = [
  { timestamp: '2024-01-01T00:00:00Z', cpuPercent: 10, memoryPercent: 40, diskPercent: 60 },
  { timestamp: '2024-01-01T00:00:30Z', cpuPercent: 20, memoryPercent: 45, diskPercent: 61 },
];

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('useSystemHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns history data from the system history endpoint', async () => {
    vi.mocked(historyServiceModule.historyService.getSystemHistory).mockResolvedValueOnce({
      data: { success: true, data: mockSnapshots, timestamp: '2024-01-01T00:00:30Z' },
    } as any);

    const { result } = renderHook(() => useSystemHistory(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.data).toEqual(mockSnapshots);
    });

    expect(historyServiceModule.historyService.getSystemHistory).toHaveBeenCalledWith(120);
  });

  it('returns empty array as data when API returns empty history', async () => {
    vi.mocked(historyServiceModule.historyService.getSystemHistory).mockResolvedValueOnce({
      data: { success: true, data: [], timestamp: '2024-01-01T00:00:00Z' },
    } as any);

    const { result } = renderHook(() => useSystemHistory(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.data).toEqual([]);
    });
  });

  it('returns isLoading true before data arrives', () => {
    vi.mocked(historyServiceModule.historyService.getSystemHistory).mockReturnValue(
      new Promise(() => {})
    );

    const { result } = renderHook(() => useSystemHistory(), { wrapper: createWrapper() });

    expect(result.current.isLoading).toBe(true);
  });
});
