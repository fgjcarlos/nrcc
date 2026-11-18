import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Stub the heavy subcomponents so the test surface is just the alert + retry
// button. Each subcomponent receives whatever props BackupsView forwards and
// renders nothing — keeps the test focused on the retry behaviour.
const { retentionMutate, retentionProps } = vi.hoisted(() => ({
  retentionMutate: vi.fn(),
  retentionProps: { current: null as null | { onSave: (manual: number, auto: number, preRestore: number) => void } },
}));

vi.mock('@/features/backups/components', () => ({
  BackupSummaryCards: () => null,
  BackupListSection: () => null,
  BackupDetailSection: () => null,
  SchedulerConfigSection: () => null,
  RetentionPolicySection: (props: { onSave: (manual: number, auto: number, preRestore: number) => void }) => {
    retentionProps.current = props;
    return null;
  },
}));

vi.mock('@/features/backups/hooks/useBackupsActions', () => ({
  useBackupsActions: () => ({
    saveConfigMutation: { isPending: false, isError: false, mutate: vi.fn(), mutateAsync: vi.fn() },
    createMutation: { isPending: false, mutate: vi.fn() },
    restoreMutation: { isPending: false, variables: null, mutate: vi.fn() },
    deleteMutation: { isPending: false, variables: null, mutate: vi.fn() },
    retentionMutation: { isPending: false, isError: false, mutate: retentionMutate, mutateAsync: vi.fn() },
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
      config: null, configLoading: false, configError: false,
      status: null, statusLoading: false, statusError: true,
      observability: null, observabilityLoading: false, observabilityError: false,
      backupList: undefined, backupsLoading: false, backupsError: false, backups: [],
      detail: null, detailLoading: false, detailFetching: false,
      storage: null, storageLoading: false, storageError: false,
      refetchConfig, refetchStatus, refetchObservability, refetchBackups, refetchStorage,
    });

    const user = userEvent.setup();
    renderWithProviders();

    const retry = await screen.findByRole('button', { name: /retry/i });
    await user.click(retry);

    expect(refetchConfig).toHaveBeenCalledTimes(1);
    expect(refetchStatus).toHaveBeenCalledTimes(1);
    expect(refetchObservability).toHaveBeenCalledTimes(1);
    expect(refetchStorage).toHaveBeenCalledTimes(1);
    expect(reloadSpy).not.toHaveBeenCalled();

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { ...window.location, reload: originalReload },
    });
  });
});

describe('BackupsView retention save', () => {
  const persistedConfig = {
    enabled: true,
    schedule: 'daily' as const,
    customSchedule: '',
    retentionManual: 30,
    retentionAuto: 30,
    retentionPreRestore: 5,
    includeConfig: true,
    includeSettings: true,
    includeFlowsCred: false,
    includePackageJson: true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    useBackupsDataMock.mockReturnValue({
      config: persistedConfig, configLoading: false, configError: false,
      status: null, statusLoading: false, statusError: false,
      observability: null, observabilityLoading: false, observabilityError: false,
      backupList: undefined, backupsLoading: false, backupsError: false, backups: [],
      detail: null, detailLoading: false, detailFetching: false,
      storage: null, storageLoading: false, storageError: false,
      refetchConfig: vi.fn(), refetchStatus: vi.fn(), refetchObservability: vi.fn(), refetchBackups: vi.fn(), refetchStorage: vi.fn(),
    });
  });

  it('sends distinct manual, auto and pre-restore retention values to the persistence call', () => {
    renderWithProviders();

    act(() => retentionProps.current?.onSave(7, 14, 3));

    expect(retentionMutate).toHaveBeenCalledTimes(1);
    expect(retentionMutate).toHaveBeenCalledWith(
      expect.objectContaining({ retentionManual: 7, retentionAuto: 14, retentionPreRestore: 3 }),
    );
  });

  it('still saves manual retention and keeps the rest of the persisted config', () => {
    renderWithProviders();

    act(() => retentionProps.current?.onSave(90, 30, 5));

    expect(retentionMutate).toHaveBeenCalledWith({ ...persistedConfig, retentionManual: 90 });
  });
});
