import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { updateService, type UpdateFlowState } from '@/features/updates/services';
import { useUpdatesData } from './useUpdatesData';

vi.mock('@/features/updates/services/updateService', () => ({
  updateService: { getStatus: vi.fn(), getFlowState: vi.fn(), getHistory: vi.fn() },
}));

const before = { currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z' };
const after = { ...before, currentVersion: '4.1.0', updateAvailable: false };

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client }, children);
  return renderHook(() => useUpdatesData(), { wrapper });
}

describe('useUpdatesData', () => {
  beforeEach(() => {
    vi.mocked(updateService.getHistory).mockResolvedValue([]);
  });

  it.each<UpdateFlowState['state']>(['Completed', 'Failed'])(
    'refreshes the status once the running update settles as %s',
    async (terminal) => {
      vi.mocked(updateService.getStatus).mockReset().mockResolvedValueOnce(before).mockResolvedValue(after);
      vi.mocked(updateService.getFlowState).mockReset()
        .mockResolvedValueOnce({ state: 'Applying', phase: 'applying' })
        .mockResolvedValue({ state: terminal, phase: 'done' });
      const { result } = setup();

      await waitFor(() => expect(result.current.flowState?.state).toBe(terminal));
      await waitFor(() => expect(result.current.status).toEqual(after));
    },
  );

  it('does not refetch the status when the page opens on an already finished update', async () => {
    vi.mocked(updateService.getStatus).mockReset().mockResolvedValue(before);
    vi.mocked(updateService.getFlowState).mockReset().mockResolvedValue({ state: 'Completed', phase: 'completed' });
    const { result } = setup();

    await waitFor(() => expect(result.current.flowState?.state).toBe('Completed'));
    await waitFor(() => expect(result.current.status).toEqual(before));
    expect(updateService.getStatus).toHaveBeenCalledTimes(1);
  });
});
