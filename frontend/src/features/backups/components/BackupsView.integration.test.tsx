import { describe, expect, it } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';
import { i18n } from '@/i18n';
import { server } from '@/test/msw/server';
import { backupConfig } from '@/test/msw/fixtures';
import type { BackupConfig } from '@/features/backups/services';
import { BackupsView } from './BackupsView';

// Real view, real hooks and services; only the network is mocked (MSW).
const ok = <T,>(data: T) => HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

/** Serves GET/POST /api/backups/config from a mutable server-side record. */
function serveConfig(initial: BackupConfig) {
  const state = { config: initial, gets: 0 };
  server.use(
    http.get('/api/backups/config', () => {
      state.gets += 1;
      return ok(state.config);
    }),
    http.post('/api/backups/config', async ({ request }) => {
      const body = (await request.json()) as BackupConfig;
      // The backend clears customSchedule for preset schedules (internal/service/backup.go normalizeBackupConfig).
      state.config = { ...body, customSchedule: body.schedule === 'custom' ? body.customSchedule : '' };
      return ok(state.config);
    }),
  );
  return state;
}

function renderView() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <BackupsView />
    </QueryClientProvider>,
  );
  return queryClient;
}

const frequency = () => screen.getByLabelText(i18n.t('backups:scheduler.frequency'));
const retention = (key: 'manual' | 'auto' | 'preRestore') => screen.getByLabelText(i18n.t(`backups:retention.${key}`));
const saved: BackupConfig = { ...backupConfig, schedule: 'weekly', enabled: true, retentionManual: 7, retentionAuto: 14, retentionPreRestore: 3 };

describe('BackupsView schedule draft against the server config', () => {
  it('shows the server frequency and retention on first load', async () => {
    serveConfig(saved);
    renderView();

    await waitFor(() => expect(frequency()).toHaveValue('weekly'));
    // RetentionEditor copies its props into local state in an effect, one commit later.
    await waitFor(() => expect(retention('manual')).toHaveValue(7));
    expect(retention('auto')).toHaveValue(14);
    expect(retention('preRestore')).toHaveValue(3);
  });

  it('keeps unsaved edits when a background refetch brings new server values', async () => {
    const state = serveConfig(saved);
    const queryClient = renderView();
    await waitFor(() => expect(frequency()).toHaveValue('weekly'));

    await userEvent.setup().selectOptions(frequency(), 'hourly');
    state.config = { ...saved, schedule: 'daily', retentionManual: 9 };
    const getsBefore = state.gets;
    await queryClient.refetchQueries({ queryKey: ['backups-config'] });

    expect(state.gets).toBeGreaterThan(getsBefore);
    expect(frequency()).toHaveValue('hourly');
    expect(retention('manual')).toHaveValue(7);
  });

  it('matches the server after a save and keeps following later refetches', async () => {
    const state = serveConfig(saved);
    const queryClient = renderView();
    await waitFor(() => expect(frequency()).toHaveValue('weekly'));

    const user = userEvent.setup();
    await user.selectOptions(frequency(), 'hourly');
    await user.click(screen.getByRole('button', { name: i18n.t('backups:scheduler.save') }));

    await waitFor(() => expect(state.config.schedule).toBe('hourly'));
    await waitFor(() => expect(screen.getByText(i18n.t('common:saved'))).toBeInTheDocument());
    expect(frequency()).toHaveValue('hourly');

    // A clean draft adopts what the server reports next.
    state.config = { ...state.config, schedule: 'daily', retentionAuto: 21 };
    await queryClient.refetchQueries({ queryKey: ['backups-config'] });
    await waitFor(() => expect(frequency()).toHaveValue('daily'));
    await waitFor(() => expect(retention('auto')).toHaveValue(21));
  });
});

describe('BackupsView backup rows', () => {
  it('shows the size the API reports for each backup', async () => {
    renderView();

    const name = await screen.findByRole('button', { name: 'Manual smoke backup' });
    const row = name.closest('tr');
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).getByText('4 KB')).toBeInTheDocument();
  });
});
