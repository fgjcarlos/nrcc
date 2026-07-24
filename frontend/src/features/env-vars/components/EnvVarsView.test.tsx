import { StrictMode } from 'react';
import { render, waitFor } from '@testing-library/react';
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
    });

    await Promise.resolve();
    expect(refetchEnvVars).not.toHaveBeenCalled();
    expect(toastError).not.toHaveBeenCalled();
  });
});
