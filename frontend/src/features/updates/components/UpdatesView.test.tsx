import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useUpdatesActions } from '@/features/updates/hooks/useUpdatesActions';
import { useUpdatesData } from '@/features/updates/hooks/useUpdatesData';
import { UpdatesView } from './UpdatesView';

vi.mock('@/features/updates/hooks/useUpdatesData');
vi.mock('@/features/updates/hooks/useUpdatesActions');

describe('UpdatesView confirmation overlay', () => {
  beforeEach(() => {
    vi.mocked(useUpdatesData).mockReturnValue({
      status: {
        currentVersion: '4.0.0',
        latestVersion: '4.1.0',
        updateAvailable: true,
        checkedAt: '2026-08-25T00:00:00Z',
        canInplaceApply: true,
        strategy: 'npm-global',
      },
      statusLoading: false,
      statusRefetch: vi.fn(),
      flowState: { state: 'Idle', phase: 'idle' },
      flowStateLoading: false,
      history: [],
      historyLoading: false,
    });
    vi.mocked(useUpdatesActions).mockReturnValue({
      checkMutation: {
        isPending: false,
        mutateAsync: vi.fn(),
      },
      applyMutation: {
        isPending: false,
        mutate: vi.fn(),
      },
    } as unknown as ReturnType<typeof useUpdatesActions>);
  });

  it('hides apply when the server does not explicitly allow in-place updates', () => {
    vi.mocked(useUpdatesData).mockReturnValue({
      status: {
        currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true,
        checkedAt: '2026-08-25T00:00:00Z', canInplaceApply: false, strategy: 'image-local',
      },
      statusLoading: false, statusRefetch: vi.fn(),
      flowState: { state: 'Idle', phase: 'idle' }, flowStateLoading: false,
      history: [], historyLoading: false,
    });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: 'Apply update' })).not.toBeInTheDocument();
    expect(screen.getByText(/whole NRCC image/i)).toBeInTheDocument();
  });

  it('fails closed when a claimed capability has an unknown current version', () => {
    vi.mocked(useUpdatesData).mockReturnValue({
      status: { currentVersion: 'unknown', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z', canInplaceApply: true, strategy: 'npm-global' },
      statusLoading: false, statusRefetch: vi.fn(),
      flowState: { state: 'Idle', phase: 'idle' }, flowStateLoading: false,
      history: [], historyLoading: false,
    });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: 'Apply update' })).not.toBeInTheDocument();
  });

  it('fails closed when old status responses omit capability fields', () => {
    vi.mocked(useUpdatesData).mockReturnValue({
      status: { currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z' },
      statusLoading: false, statusRefetch: vi.fn(),
      flowState: { state: 'Idle', phase: 'idle' }, flowStateLoading: false,
      history: [], historyLoading: false,
    });
    render(<UpdatesView />);
    expect(screen.queryByRole('button', { name: 'Apply update' })).not.toBeInTheDocument();
  });

  it('localizes a known backend update failure code', () => {
    vi.mocked(useUpdatesData).mockReturnValue({
      status: { currentVersion: '4.0.0', latestVersion: '4.1.0', updateAvailable: true, checkedAt: '2026-08-25T00:00:00Z', canInplaceApply: false, strategy: 'image-local' },
      statusLoading: false, statusRefetch: vi.fn(),
      flowState: { state: 'Failed', phase: 'applying', error: 'image_unsupported' }, flowStateLoading: false,
      history: [], historyLoading: false,
    });
    render(<UpdatesView />);
    expect(screen.getByText('Update and redeploy the whole NRCC image with persistent data to upgrade Node-RED.')).toBeInTheDocument();
    expect(screen.queryByText('image_unsupported')).not.toBeInTheDocument();
  });

  it('opens the update confirmation above the Updates page overlay context (#721)', async () => {
    const user = userEvent.setup();
    const { container } = render(<UpdatesView />);

    await user.click(screen.getByRole('button', { name: 'Apply update' }));

    const dialog = screen.getByRole('dialog', { name: 'Update Node-RED' });
    expect(container).not.toContainElement(dialog);
    expect(dialog.closest('[data-confirmation-dialog-portal]')?.parentElement).toBe(document.body);
  });
});
