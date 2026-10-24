/**
 * useConfigurationActions — issue #880.
 *
 * A raw save rewrites settings.js, and the backend reconciles the
 * structured configuration with the applied source (issue #880). The
 * structured configuration query therefore has to be refreshed after a
 * raw save; otherwise the operator keeps seeing the pre-save
 * configuration and the next visual save would carry stale values.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

const mocks = vi.hoisted(() => ({
  saveRaw: vi.fn(),
  updateConfig: vi.fn(),
}));

vi.mock('../services', () => ({
  configService: { updateConfig: mocks.updateConfig },
  settingsService: { saveRaw: mocks.saveRaw },
}));

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useConfigurationActions } from './useConfigurationActions';
import { queryKeys } from '@/shared/lib/queryKeys';

function renderActions() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
  const wrapper = ({ children }: any) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
  const view = renderHook(() => useConfigurationActions(), { wrapper });
  return { ...view, queryClient, invalidateSpy };
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.saveRaw.mockResolvedValue({ data: { data: { content: '' }, success: true, timestamp: '' } });
});

describe('useConfigurationActions raw save', () => {
  it('refreshes the structured configuration query after a raw save', async () => {
    const { result, invalidateSpy } = renderActions();

    await act(async () => {
      result.current.saveRawSettingsMutation.mutate({
        content: 'module.exports = { uiPort: 7777 };\n',
        expectedRevision: 'rev-1',
      });
    });

    await waitFor(() => {
      expect(mocks.saveRaw).toHaveBeenCalledWith('module.exports = { uiPort: 7777 };\n', 'rev-1');
    });

    const invalidated = invalidateSpy.mock.calls.map((call) => call[0]);
    expect(invalidated).toContainEqual({ queryKey: queryKeys.config.root });
    expect(invalidated).toContainEqual({ queryKey: queryKeys.config.rawSettings });
  });

  it('sends the revision the operator read', async () => {
    const { result } = renderActions();

    await act(async () => {
      result.current.saveRawSettingsMutation.mutate({
        content: 'module.exports = {};\n',
        expectedRevision: 'rev-42',
      });
    });

    await waitFor(() => {
      expect(mocks.saveRaw).toHaveBeenCalledWith('module.exports = {};\n', 'rev-42');
    });
  });
});
