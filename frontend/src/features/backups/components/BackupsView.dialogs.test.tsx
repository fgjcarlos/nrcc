import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { i18n } from '@/i18n';
import enBackups from '@/locales/en/backups.json';
import esBackups from '@/locales/es/backups.json';
import type { BackupSummary } from '@/features/backups/services';

// Public render of the whole /backups view with its real subcomponents;
// only the data and mutation hooks are stubbed.
const restoreMutate = vi.fn();
const deleteMutate = vi.fn();
const createMutate = vi.fn();
vi.mock('@/features/backups/hooks/useBackupsActions', () => ({
  useBackupsActions: () => ({
    saveConfigMutation: { isPending: false, isError: false, error: null, mutate: vi.fn() },
    createMutation: { isPending: false, mutate: createMutate },
    restoreMutation: { isPending: false, variables: undefined, mutate: restoreMutate },
    deleteMutation: { isPending: false, variables: undefined, mutate: deleteMutate },
    retentionMutation: { isPending: false, isError: false, mutate: vi.fn() },
  }),
}));

const useBackupsDataMock = vi.fn();
vi.mock('@/features/backups/hooks/useBackupsData', () => ({
  useBackupsData: (...args: unknown[]) => useBackupsDataMock(...args),
}));

import { BackupsView } from './BackupsView';

const backup: BackupSummary = {
  id: '3f2b9c1e-8a7d-4e6f-9b0a-1c2d3e4f5a6b',
  name: 'Before upgrade',
  type: 'manual',
  createdAt: '2026-01-01T12:00:00.000Z',
  triggeredBy: 'manual',
  fileCount: 4,
  totalSize: 4096,
};

function mockData(backups: BackupSummary[], overrides: Record<string, unknown> = {}) {
  useBackupsDataMock.mockReturnValue({
    config: undefined, status: undefined, statusError: false,
    observability: undefined, observabilityError: false,
    backupList: { items: backups, total: backups.length, page: 1, limit: 10 },
    backupsLoading: false, backupsError: false, backups,
    detail: undefined, detailLoading: false, detailFetching: false,
    storage: undefined,
    refetchConfig: vi.fn(), refetchStatus: vi.fn(), refetchObservability: vi.fn(),
    refetchBackups: vi.fn(), refetchStorage: vi.fn(),
    ...overrides,
  });
}

const renderView = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <BackupsView />
    </QueryClientProvider>
  );

const expectedDate = new Date(backup.createdAt).toLocaleString('en', { dateStyle: 'medium', timeStyle: 'short' });

/** Spanish strings the EN render must never show: every ES catalog value that differs from EN, plus old literals. */
function spanishLeaks(text: string): string[] {
  const flatten = (value: unknown): string[] =>
    typeof value === 'string' ? [value] : Object.values(value as object).flatMap(flatten);
  const en = new Set(flatten(enBackups));
  const es = flatten(esBackups).filter((value) => !en.has(value) && value.length > 3 && !value.includes('{{'));
  return [...es, 'Sin programar', 'Programado', 'Requiere atención', 'Restaurar', 'Fecha no disponible', '¿']
    .filter((value) => text.includes(value));
}

/** Words that look like an unresolved i18n key (camelCase or dotted), checked per text node. */
function rawKeys(): string[] {
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  const words: string[] = [];
  for (let node = walker.nextNode(); node; node = walker.nextNode()) words.push(...(node.textContent ?? '').split(/\s+/));
  return words.filter(
    (word) =>
      /^(?:common|backups):/.test(word) ||
      /^[a-z]+[A-Z][A-Za-z]*$/.test(word) ||
      (/^[a-z][A-Za-z]*(?:\.[A-Za-z-]+)+$/.test(word) && !/\.(?:js|json)$/.test(word))
  );
}

describe('BackupsView restore and delete dialogs', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await i18n.changeLanguage('en');
  });
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('restore shows the real steps, the backup identity and a short token', async () => {
    mockData([backup]);
    const user = userEvent.setup();
    renderView();

    await user.click(screen.getByRole('button', { name: `Restore backup from ${expectedDate}` }));
    const dialog = screen.getByRole('dialog', { name: 'Restore backup' });

    const steps = within(dialog).getAllByRole('listitem').map((item) => item.textContent);
    expect(steps).toHaveLength(4);
    expect(steps[0]).toMatch(/Node-RED stops/);
    expect(steps[1]).toMatch(/pre-restore snapshot/);
    expect(steps[2]).toMatch(/replace the current ones/);
    expect(steps[2]).toMatch(/settings\.js/);
    expect(steps[3]).toMatch(/Node-RED starts again/);
    expect(steps[3]).toMatch(/No readiness check/);

    expect(within(dialog).getByText('Before upgrade')).toBeInTheDocument();
    expect(within(dialog).getByText(expectedDate)).toBeInTheDocument();
    expect(within(dialog).getByText('Manual')).toBeInTheDocument();
    expect(within(dialog).getByText('4 KB')).toBeInTheDocument();
    // The token is a word, never the 36-character id.
    expect(within(dialog).getByLabelText('Type "restore" to confirm')).toBeInTheDocument();
    expect(within(dialog).queryByText(new RegExp(backup.id))).toBeNull();
  });

  it('keeps confirm disabled until the token matches and names the backup on the button', async () => {
    mockData([backup]);
    const user = userEvent.setup();
    renderView();

    await user.click(screen.getByRole('button', { name: `Restore backup from ${expectedDate}` }));
    const dialog = screen.getByRole('dialog', { name: 'Restore backup' });
    const confirm = within(dialog).getByRole('button', { name: `Restore ${expectedDate}` });
    const token = within(dialog).getByLabelText('Type "restore" to confirm');

    expect(token).toHaveFocus();
    expect(confirm).toBeDisabled();
    await user.type(token, 'restor{Enter}');
    expect(confirm).toBeDisabled();
    expect(restoreMutate).not.toHaveBeenCalled();

    await user.type(token, 'e');
    expect(confirm).toBeEnabled();
    await user.click(confirm);
    expect(restoreMutate).toHaveBeenCalledTimes(1);
    expect(restoreMutate).toHaveBeenCalledWith(backup.id);
    expect(deleteMutate).not.toHaveBeenCalled();
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('Enter confirms once the token matches', async () => {
    mockData([backup]);
    const user = userEvent.setup();
    renderView();

    await user.click(screen.getByRole('button', { name: `Restore backup from ${expectedDate}` }));
    await user.type(screen.getByLabelText('Type "restore" to confirm'), 'restore{Enter}');
    expect(restoreMutate).toHaveBeenCalledTimes(1);
    expect(restoreMutate).toHaveBeenCalledWith(backup.id);
  });

  it('delete is a distinct danger dialog with its own token and no restore steps', async () => {
    mockData([backup]);
    const user = userEvent.setup();
    renderView();

    await user.click(screen.getByRole('button', { name: `Delete backup from ${expectedDate}` }));
    const dialog = screen.getByRole('dialog', { name: 'Delete backup' });
    expect(within(dialog).queryByRole('listitem')).toBeNull();
    expect(within(dialog).getByText(/cannot be undone/)).toBeInTheDocument();

    const confirm = within(dialog).getByRole('button', { name: `Delete ${expectedDate}` });
    expect(confirm).toHaveClass('action-btn-danger');
    expect(confirm).toBeDisabled();
    await user.type(within(dialog).getByLabelText('Type "delete" to confirm'), 'restore');
    expect(confirm).toBeDisabled();
    await user.clear(within(dialog).getByLabelText('Type "delete" to confirm'));
    await user.type(within(dialog).getByLabelText('Type "delete" to confirm'), 'delete');
    await user.click(confirm);
    expect(deleteMutate).toHaveBeenCalledWith(backup.id);
    expect(restoreMutate).not.toHaveBeenCalled();
  });

  it('restore uses the Spanish token and copy in an es render', async () => {
    await i18n.changeLanguage('es');
    mockData([backup]);
    const user = userEvent.setup();
    renderView();

    const esDate = new Date(backup.createdAt).toLocaleString('es', { dateStyle: 'medium', timeStyle: 'short' });
    await user.click(screen.getByRole('button', { name: `Restaurar respaldo del ${esDate}` }));
    const dialog = screen.getByRole('dialog', { name: 'Restaurar respaldo' });
    expect(within(dialog).getByLabelText('Escribe "restaurar" para confirmar')).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: `Restaurar ${esDate}` })).toBeDisabled();
  });

  it('an en render shows no Spanish and no raw keys, with the list, the empty state and the dialog', async () => {
    mockData([backup], {
      status: {
        enabled: false, scheduled: false, schedule: 'disabled', customSchedule: '', activeSpec: '',
        nextRunAt: '', lastRunAt: '', lastSuccessAt: '',
      },
    });
    const user = userEvent.setup();
    const { unmount } = renderView();
    expect(screen.getByText('Not scheduled', { selector: '[role="status"] *' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: `Restore backup from ${expectedDate}` }));
    const withDialog = document.body.textContent ?? '';
    expect(spanishLeaks(withDialog)).toEqual([]);
    expect(rawKeys()).toEqual([]);
    unmount();

    mockData([]);
    renderView();
    expect(screen.getByRole('button', { name: 'Create first backup' })).toBeInTheDocument();
    const empty = document.body.textContent ?? '';
    expect(spanishLeaks(empty)).toEqual([]);
    expect(rawKeys()).toEqual([]);
  });
});
