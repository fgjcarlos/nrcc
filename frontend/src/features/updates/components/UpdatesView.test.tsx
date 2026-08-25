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

  it('opens the update confirmation above the Updates page overlay context (#721)', async () => {
    const user = userEvent.setup();
    const { container } = render(<UpdatesView />);

    await user.click(screen.getByRole('button', { name: 'Apply update' }));

    const dialog = screen.getByRole('dialog', { name: 'Update Node-RED' });
    expect(container).not.toContainElement(dialog);
    expect(dialog.closest('[data-confirmation-dialog-portal]')?.parentElement).toBe(document.body);
  });
});
