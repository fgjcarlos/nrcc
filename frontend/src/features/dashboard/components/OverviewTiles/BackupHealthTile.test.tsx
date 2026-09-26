import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { BackupHealthTile } from './BackupHealthTile';
import type { BackupObservability } from '@/features/backups/services/backupService';

function buildBackups(overrides: Partial<BackupObservability> = {}): BackupObservability {
  return {
    scheduler: {
      enabled: true,
      scheduled: true,
      schedule: 'daily',
      customSchedule: '',
      activeSpec: '0 2 * * *',
      nextRunAt: '2026-01-02T02:00:00.000Z',
      lastRunAt: '2026-01-01T02:00:00.000Z',
      lastSuccessAt: '2026-01-01T02:00:00.000Z',
      lastBackupId: 'backup-001',
    },
    storage: {
      totalBackups: 12,
      totalSize: 8_589_934_592,
      manualCount: 4,
      autoCount: 8,
    },
    recentEvents: [],
    ...overrides,
  } as BackupObservability;
}

function renderTile(props: Parameters<typeof BackupHealthTile>[0]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <BackupHealthTile {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('BackupHealthTile', () => {
  it('renders the success palette when the scheduler is healthy', () => {
    renderTile({ backups: buildBackups() });
    const tile = screen.getByTestId('overview-backup-health-tile');
    expect(tile).toBeInTheDocument();
    // The header chip carries success when scheduled && !lastError.
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
  });

  it('renders the warning palette when the scheduler has a last error', () => {
    renderTile({
      backups: buildBackups({
        scheduler: {
          ...buildBackups().scheduler,
          lastError: 'disk full',
        },
      }),
    });
    const tile = screen.getByTestId('overview-backup-health-tile');
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
  });

  it('renders the danger palette when the scheduler is not scheduled', () => {
    renderTile({
      backups: buildBackups({
        scheduler: { ...buildBackups().scheduler, scheduled: false },
      }),
    });
    const tile = screen.getByTestId('overview-backup-health-tile');
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-danger/);
  });

  it('exposes the last error message when the scheduler reports one', () => {
    renderTile({
      backups: buildBackups({
        scheduler: {
          ...buildBackups().scheduler,
          lastError: 'permission denied',
        },
      }),
    });
    expect(screen.getByTestId('overview-backup-last-error')).toHaveTextContent(/permission denied/);
  });

  it('omits the last-error block when the scheduler is healthy', () => {
    renderTile({ backups: buildBackups() });
    expect(screen.queryByTestId('overview-backup-last-error')).not.toBeInTheDocument();
  });

  it('surfaces the deep-link to /backups', () => {
    renderTile({ backups: buildBackups() });
    const link = screen.getByTestId('overview-backup-health-link');
    expect(link).toHaveAttribute('href', '/backups');
  });

  it('renders the three summary panels even when no backups exist', () => {
    renderTile({
      backups: buildBackups({
        latestBackup: undefined,
        scheduler: { ...buildBackups().scheduler, lastBackupId: undefined, lastSuccessAt: '' },
      }),
    });
    expect(screen.getByTestId('overview-backup-last-name')).toBeInTheDocument();
    expect(screen.getByTestId('overview-backup-last-automatic')).toBeInTheDocument();
    expect(screen.getByTestId('overview-backup-storage')).toBeInTheDocument();
  });
});
