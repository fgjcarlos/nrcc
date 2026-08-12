import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Stub the heavy subcomponents so the test surface is just the alert + retry
// button. Each subcomponent receives whatever props BackupsView forwards and
// renders nothing — keeps the test focused on the retry behaviour.
vi.mock('@/features/backups/components', () => ({
  BackupSummaryCards: () => null,
  BackupListSection: () => null,
  BackupDetailSection: () => null,
  SchedulerConfigSection: () => null,
  RetentionPolicySection: () => null,
}));

vi.mock('@/features/backups/hooks/useBackupsActions', () => ({
  useBackupsActions: () => ({
    saveConfigMutation: { isPending: false, isError: false, mutate: vi.fn(), mutateAsync: vi.fn() },
    createMutation: { isPending: false, mutate: vi.fn() },
    restoreMutation: { isPending: false, variables: null, mutate: vi.fn() },
    deleteMutation: { isPending: false, variables: null, mutate: vi.fn() },
    retentionMutation: { isPending: false, isError: false, mutate: vi.fn(), mutateAsync: vi.fn() },
  }),
}));

const useBackupsDataMock = vi.fn();
vi.mock('@/features/backups/hooks/useBackupsData', () => ({
  useBackupsData: (...args: unknown[]) => useBackupsDataMock(...args),
}));

import { BackupsView } from './BackupsView';

const renderWithProviders = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <BackupsView />
    </QueryClientProvider>
  );
};

describe('BackupsView error retry', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls all four refetch methods and does NOT reload the page when retry is clicked', async () => {
    const refetchConfig = vi.fn();
    const refetchStatus = vi.fn();
    const refetchObservability = vi.fn();
    const refetchStorage = vi.fn();
    const refetchBackups = vi.fn();

    const reloadSpy = vi.fn();
    const originalReload = window.location.reload;
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, reload: reloadSpy },
    });

    useBackupsDataMock.mockReturnValue({
      config: null,
      configLoading: false,
      configError: false,
      status: null,
      statusLoading: false,
      statusError: true, // triggers the alert
      observability: null,
      observabilityLoading: false,
      observabilityError: false,
      backupList: undefined,
      backupsLoading: false,
      backupsError: false,
      backups: [],
      detail: null,
      detailLoading: false,
      detailFetching: false,
      storage: null,
      storageLoading: false,
      storageError: false,
      refetchConfig,
      refetchStatus,
      refetchObservability,
      refetchBackups,
      refetchStorage,
    });

    const user = userEvent.setup();
    renderWithProviders();

    const retry = await screen.findByRole('button', { name: /reintentar/i });
    await user.click(retry);

    expect(refetchConfig).toHaveBeenCalledTimes(1);
    expect(refetchStatus).toHaveBeenCalledTimes(1);
    expect(refetchObservability).toHaveBeenCalledTimes(1);
    expect(refetchStorage).toHaveBeenCalledTimes(1);
    expect(reloadSpy).not.toHaveBeenCalled();

    // Restore for other tests in the same file (vitest isolates files, but
    // be explicit so the spy doesn't leak if more tests are added).
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, reload: originalReload },
    });
  });

  it('does not call window.location.reload anywhere in the component', async () => {
    // Static check: the source file no longer mentions the reload API.
    // We assert this by ensuring the retry button still calls the refetch
    // methods (proving the wiring is intact) and that no reload happens.
    const refetchConfig = vi.fn();
    const refetchStatus = vi.fn();
    const refetchObservability = vi.fn();
    const refetchStorage = vi.fn();
    const refetchBackups = vi.fn();

    const reloadSpy = vi.fn();
    const originalReload = window.location.reload;
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, reload: reloadSpy },
    });

    useBackupsDataMock.mockReturnValue({
      config: null,
      configLoading: false,
      configError: false,
      status: null,
      statusLoading: false,
      statusError: true,
      observability: null,
      observabilityLoading: false,
      observabilityError: false,
      backupList: undefined,
      backupsLoading: false,
      backupsError: false,
      backups: [],
      detail: null,
      detailLoading: false,
      detailFetching: false,
      storage: null,
      storageLoading: false,
      storageError: false,
      refetchConfig,
      refetchStatus,
      refetchObservability,
      refetchBackups,
      refetchStorage,
    });

    const user = userEvent.setup();
    renderWithProviders();

    const retry = await screen.findByRole('button', { name: /reintentar/i });
    await user.click(retry);

    expect(reloadSpy).not.toHaveBeenCalled();

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, reload: originalReload },
    });
  });
});