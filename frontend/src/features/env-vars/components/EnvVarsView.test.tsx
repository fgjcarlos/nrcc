import { StrictMode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { EnvVarsView } from './EnvVarsView';

const { importFromNodeRed, refetchEnvVars, toastError } = vi.hoisted(() => ({
  importFromNodeRed: vi.fn(),
  refetchEnvVars: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock('@/features/env-vars/services', () => ({
  envService: { importFromNodeRed },
}));

vi.mock('@/features/env-vars/hooks', () => ({
  useEnvVarsData: () => ({ envVars: [], isLoading: false, refetchEnvVars }),
  useEnvVarsActions: () => ({
    createMutation: { mutate: vi.fn(), isPending: false },
    deleteMutation: { mutate: vi.fn(), isPending: false },
  }),
}));

vi.mock('@/features/auth/hooks', () => ({
  useAuth: () => ({ user: { role: 'admin' } }),
}));

vi.mock('sonner', () => ({ toast: { error: toastError } }));

describe('EnvVarsView', () => {
  beforeEach(() => {
    importFromNodeRed.mockReset();
    refetchEnvVars.mockReset();
    toastError.mockReset();
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
