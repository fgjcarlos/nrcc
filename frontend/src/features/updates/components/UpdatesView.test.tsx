import { render, screen, within } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useUpdatesActions } from '@/features/updates/hooks/useUpdatesActions';
import { useUpdatesData } from '@/features/updates/hooks/useUpdatesData';
import type { UpdateFlowState, UpdateHistoryEntry, UpdateStatus } from '@/features/updates/services';
import { UpdatesView } from './UpdatesView';

vi.mock('@/features/updates/hooks/useUpdatesData');
vi.mock('@/features/updates/hooks/useUpdatesActions');

const npmStatus: UpdateStatus = {
  currentVersion: '4.0.0',
  latestVersion: '4.1.0',
  updateAvailable: true,
  checkedAt: '2026-08-25T00:00:00Z',
  canInplaceApply: true,
  strategy: 'npm-global',
};

function mockData(
  status: UpdateStatus | undefined,
  overrides: {
    flowState?: UpdateFlowState;
    statusLoading?: boolean;
    history?: UpdateHistoryEntry[];
    historyLoading?: boolean;
  } = {},
) {
  vi.mocked(useUpdatesData).mockReturnValue({
    status,
    statusLoading: overrides.statusLoading ?? false,
    statusRefetch: vi.fn(),
    flowState: overrides.flowState ?? { state: 'Idle', phase: 'idle' },
    flowStateLoading: false,
    history: overrides.history ?? [],
    historyLoading: overrides.historyLoading ?? false,
  } as unknown as ReturnType<typeof useUpdatesData>);
}

function mockActions(checkPending = false) {
  const mutate = vi.fn();
  vi.mocked(useUpdatesActions).mockReturnValue({
    checkMutation: { isPending: checkPending, mutateAsync: vi.fn() },
    applyMutation: { isPending: false, mutate },
  } as unknown as ReturnType<typeof useUpdatesActions>);
  return mutate;
}

describe('UpdatesView', () => {
  beforeEach(() => {
    mockData(npmStatus);
    mockActions();
  });

  it('titles the page without a kicker above the heading', () => {
    render(<UpdatesView />);
    expect(screen.getByRole('heading', { level: 1, name: 'Updates' })).toBeInTheDocument();
    expect(screen.queryByText('Maintenance')).not.toBeInTheDocument();
  });

  it('shows installed and latest versions as mono designators', () => {
    render(<UpdatesView />);
    expect(screen.getByText('4.0.0')).toHaveClass('font-mono');
    expect(screen.getByText('4.1.0')).toHaveClass('font-mono');
  });

  it.each([
    ['up to date', { ...npmStatus, latestVersion: '4.0.0', updateAvailable: false }, false, 'Up to date'],
    ['update available', npmStatus, false, 'Update available'],
    ['checking', npmStatus, true, 'Checking for updates…'],
    ['check failed', { ...npmStatus, updateAvailable: false, error: 'Network error while checking for updates.' }, false, 'Check failed'],
    ['unconfirmed', { ...npmStatus, latestVersion: '', updateAvailable: false }, false, 'Not confirmed'],
  ])('states the %s check result', (_label, status, checking, copy) => {
    mockData(status as UpdateStatus);
    mockActions(checking as boolean);
    render(<UpdatesView />);
    expect(screen.getByRole('status', { name: copy as string })).toBeInTheDocument();
  });

  it('shows the server reason when the check failed and keeps the update disabled', () => {
    mockData({ ...npmStatus, error: 'Network error while checking for updates.' });
    render(<UpdatesView />);
    expect(screen.getByText('Network error while checking for updates.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Update to/ })).not.toBeInTheDocument();
  });

  it('hides the update action when the server does not explicitly allow in-place updates', () => {
    mockData({ ...npmStatus, canInplaceApply: false, strategy: 'image-local' });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: /^Update to/ })).not.toBeInTheDocument();
    expect(screen.getByText(/whole NRCC image/i)).toBeInTheDocument();
  });

  it('fails closed when a claimed capability has an unknown current version', () => {
    mockData({ ...npmStatus, currentVersion: 'unknown' });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: /^Update to/ })).not.toBeInTheDocument();
  });

  it('fails closed when old status responses omit capability fields', () => {
    mockData({ currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z' });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: /^Update to/ })).not.toBeInTheDocument();
  });

  it('localizes a known backend update failure code', () => {
    mockData({ ...npmStatus, canInplaceApply: false, strategy: 'image-local' }, {
      flowState: { state: 'Failed', phase: 'applying', error: 'image_unsupported' },
    });
    render(<UpdatesView />);
    expect(screen.getByText('Update and redeploy the whole NRCC image with persistent data to upgrade Node-RED.')).toBeInTheDocument();
    expect(screen.queryByText('image_unsupported')).not.toBeInTheDocument();
  });

  it('says nothing was installed when the pre-update backup failed', () => {
    mockData(npmStatus, { flowState: { state: 'Failed', phase: 'backup', error: 'backup_failed' } });
    render(<UpdatesView />);
    expect(screen.getByText('The pre-update backup could not be written, so nothing was installed.')).toBeInTheDocument();
  });

  it('says a completed update still needs a Node-RED restart', () => {
    mockData(npmStatus, { flowState: { state: 'Completed', phase: 'completed', backupId: 'abcdef1234567890' } });
    render(<UpdatesView />);
    expect(screen.getByText('Node-RED 4.1.0 is installed')).toBeInTheDocument();
    expect(screen.getByText(/keeps running the previous version until it restarts/)).toBeInTheDocument();
  });

  it('confirms the update with what it really does before calling the API', async () => {
    const user = userEvent.setup();
    const mutate = mockActions();
    const { container } = render(<UpdatesView />);

    await user.click(screen.getByRole('button', { name: 'Update to 4.1.0' }));

    const dialog = screen.getByRole('dialog', { name: 'Update Node-RED to 4.1.0?' });
    // #721: the confirmation renders above the page overlay context.
    expect(container).not.toContainElement(dialog);
    expect(dialog.closest('[data-confirmation-dialog-portal]')?.parentElement).toBe(document.body);
    expect(dialog).toHaveTextContent('NRCC backs up the Node-RED data directory, then runs npm install -g node-red@4.1.0');
    expect(dialog).toHaveTextContent('Node-RED keeps running 4.0.0 until you restart it.');
    expect(mutate).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole('button', { name: 'Confirm' }));
    expect(mutate).toHaveBeenCalledTimes(1);
  });

  it('loads history with the shared skeleton', () => {
    mockData(npmStatus, { historyLoading: true });
    const { container } = render(<UpdatesView />);
    expect(container.querySelector('.skeleton-dark')).toBeNull();
    expect(screen.getByRole('status', { name: 'Loading...' })).toHaveAttribute('aria-busy', 'true');
  });

  it('lists each update attempt with its source version and backup, without a false error', () => {
    mockData(npmStatus, {
      history: [
        { id: 'a1b2c3d4e5f6', timestamp: '2026-08-20T10:00:00Z', fromVersion: '3.1.9', status: 'completed' } as unknown as UpdateHistoryEntry,
      ],
    });
    render(<UpdatesView />);
    const table = screen.getByRole('table', { name: 'Update attempts' });
    expect(within(table).getByText('3.1.9')).toHaveClass('font-mono');
    expect(within(table).getByText('a1b2c3d4')).toHaveClass('font-mono');
    expect(within(table).queryByText('Error')).not.toBeInTheDocument();
  });
});
