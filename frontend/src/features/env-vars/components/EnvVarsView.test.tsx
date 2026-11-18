import { StrictMode } from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { EnvVarsView } from './EnvVarsView';

const { importFromNodeRed, refetchEnvVars, toastError, dataState } = vi.hoisted(() => ({
  importFromNodeRed: vi.fn(),
  refetchEnvVars: vi.fn(),
  toastError: vi.fn(),
  dataState: { isLoading: false },
}));

vi.mock('@/features/env-vars/services', () => ({
  envService: { importFromNodeRed },
}));

vi.mock('@/features/env-vars/hooks', () => ({
  useEnvVarsData: () => ({ envVars: [], isLoading: dataState.isLoading, refetchEnvVars }),
  useEnvVarsActions: () => ({
    createMutation: { mutate: vi.fn(), isPending: false },
    deleteMutation: { mutate: vi.fn(), isPending: false },
  }),
}));

vi.mock('@/features/auth/hooks', () => ({
  useAuth: () => ({ user: { role: 'admin' } }),
}));

// The .env editor fetches through React Query; the tab switch only needs its panel.
vi.mock('./DotenvEditor', () => ({ DotenvEditor: () => <p>dotenv editor</p> }));

vi.mock('sonner', () => ({ toast: { error: toastError } }));

describe('EnvVarsView', () => {
  beforeEach(() => {
    importFromNodeRed.mockReset();
    refetchEnvVars.mockReset();
    toastError.mockReset();
    dataState.isLoading = false;
    importFromNodeRed.mockResolvedValue({ lines: [], issues: [], valid: true, summary: '', status: 'synced' });
  });

  it('switches between the configured table and the .env file through a labelled tablist', async () => {
    const user = userEvent.setup();
    render(<EnvVarsView />);

    const tablist = screen.getByRole('tablist', { name: 'Environment variable views' });
    const [configured, dotenv] = within(tablist).getAllByRole('tab');
    expect(configured).toHaveAccessibleName('Configured');
    expect(configured).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tabpanel', { name: 'Configured' })).toHaveTextContent('No environment variables configured');

    configured.focus();
    await user.keyboard('{ArrowRight}');
    expect(dotenv).toHaveFocus();
    expect(dotenv).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tabpanel', { name: '.env file' })).toHaveTextContent('dotenv editor');
    await waitFor(() => expect(importFromNodeRed).toHaveBeenCalled());
  });

  it('opens on the heading and a one-line intro, without a kicker', async () => {
    render(<EnvVarsView />);

    const heading = screen.getByRole('heading', { level: 1, name: 'Environment variables' });
    expect(heading.previousElementSibling).toBeNull();
    expect(screen.queryByText('Secrets')).toBeNull();
    expect(screen.getByText('Variables Node-RED runs with. Any change restarts Node-RED if it is running.')).toBeInTheDocument();
    await waitFor(() => expect(importFromNodeRed).toHaveBeenCalled());
  });

  it('announces loading through the shared status instead of a table row', async () => {
    dataState.isLoading = true;
    render(<EnvVarsView />);

    const panel = screen.getByRole('tabpanel', { name: 'Configured' });
    expect(within(panel).getByRole('status')).toHaveTextContent('Loading...');
    expect(within(panel).queryByRole('table')).toBeNull();
    await waitFor(() => expect(importFromNodeRed).toHaveBeenCalled());
  });

  it('imports Node-RED globals once on mount and refreshes the list', async () => {
    importFromNodeRed.mockResolvedValue({
      lines: [{ line: 1, key: 'FROM_NODE_RED', value: 'yes', type: 'string' }],
      issues: [],
      valid: true,
      summary: '1 variable(s) ready',
      status: 'synced',
    });

    render(
      <StrictMode>
        <EnvVarsView />
      </StrictMode>,
    );

    await waitFor(() => expect(importFromNodeRed).toHaveBeenCalledTimes(1));
    expect(importFromNodeRed).toHaveBeenCalledWith(true);
    await waitFor(() => expect(refetchEnvVars).toHaveBeenCalledTimes(1));
  });

  it('ignores a completed import after unmount', async () => {
    let resolveImport!: (result: {
      lines: { line: number; key: string; value: string; type: string }[];
      issues: never[];
      valid: boolean;
      summary: string;
      status: 'synced';
    }) => void;
    importFromNodeRed.mockReturnValue(new Promise((resolve) => {
      resolveImport = resolve;
    }));

    const { unmount } = render(<EnvVarsView />);
    unmount();
    resolveImport({
      lines: [{ line: 1, key: 'FROM_NODE_RED', value: 'yes', type: 'string' }],
      issues: [],
      valid: true,
      summary: '1 variable(s) ready',
      status: 'synced',
    });

    await Promise.resolve();
    expect(refetchEnvVars).not.toHaveBeenCalled();
    expect(toastError).not.toHaveBeenCalled();
  });

  it('shows an inline unavailable state and retains the manual resync action', async () => {
    importFromNodeRed.mockResolvedValue({
      lines: [],
      issues: [{ line: 0, reason: 'managed runtime disabled' }],
      valid: false,
      summary: 'managed runtime disabled',
      status: 'unavailable',
    });

    render(<EnvVarsView />);

    expect(await screen.findByTestId('node-red-sync-status')).toHaveTextContent('Node-RED synchronization is unavailable');
    await screen.getByRole('button', { name: 'Resync Node-RED' }).click();
    await waitFor(() => expect(importFromNodeRed).toHaveBeenCalledTimes(2));
  });

  it('shows synchronization progress and errors inline', async () => {
    let resolveImport!: (result: {
      lines: never[];
      issues: { line: number; reason: string }[];
      valid: boolean;
      summary: string;
      status: 'error';
    }) => void;
    importFromNodeRed.mockReturnValue(new Promise((resolve) => {
      resolveImport = resolve;
    }));

    render(<EnvVarsView />);

    expect(await screen.findByTestId('node-red-sync-status')).toHaveTextContent('Synchronizing Node-RED environment entries');
    resolveImport({
      lines: [],
      issues: [{ line: 0, reason: 'flows.json is malformed' }],
      valid: false,
      summary: 'flows.json is malformed',
      status: 'error',
    });
    expect(await screen.findByTestId('node-red-sync-status')).toHaveTextContent('Could not synchronize Node-RED environment entries');
  });
});
