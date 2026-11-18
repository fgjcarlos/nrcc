import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EnvVarRow } from './EnvVarRow';
import type { EnvVar } from '../services/envService';

function renderRow(envVar: EnvVar, { showSecret = false, onToggleSecret = vi.fn() } = {}) {
  return render(
    <table>
      <tbody>
        <EnvVarRow
          envVar={envVar}
          onDelete={vi.fn()}
          onEdit={vi.fn()}
          onToggleSecret={onToggleSecret}
          showSecret={showSecret}
        />
      </tbody>
    </table>,
  );
}

describe('EnvVarRow', () => {
  it('marks secrets as runtime-only', () => {
    renderRow({ key: 'API_KEY', value: '', type: 'secret', encrypted: true });

    expect(screen.getByText('runtime only')).toBeInTheDocument();
  });

  it('does not mark non-secret variables as runtime-only', () => {
    renderRow({ key: 'PUBLIC_URL', value: 'https://example.test', type: 'string' });

    expect(screen.queryByText('runtime only')).not.toBeInTheDocument();
  });

  it('shows whether a variable originated in NRCC or Node-RED', () => {
    renderRow({ key: 'FROM_NODE_RED', value: 'yes', type: 'string', source: 'node-red' });

    expect(screen.getByText('Node-RED')).toBeInTheDocument();
  });

  it('names the secret toggle by its action and variable and exposes its pressed state', async () => {
    const onToggleSecret = vi.fn();
    const secret: EnvVar = { key: 'API_KEY', value: 's3cret', type: 'secret', encrypted: true };
    const { unmount } = renderRow(secret, { onToggleSecret });

    const show = screen.getByRole('button', { name: 'Show value of API_KEY' });
    expect(show).toHaveAttribute('aria-pressed', 'false');
    expect(screen.queryByText('s3cret')).toBeNull();
    await userEvent.click(show);
    expect(onToggleSecret).toHaveBeenCalledWith('API_KEY');
    unmount();

    renderRow(secret, { showSecret: true });
    expect(screen.getByRole('button', { name: 'Hide value of API_KEY' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('s3cret')).toBeInTheDocument();
  });

  it('renders type and source as neutral status chips', () => {
    renderRow({ key: 'API_KEY', value: '', type: 'secret', encrypted: true, source: 'node-red' });

    for (const name of ['secret', 'Node-RED']) {
      const chip = screen.getByRole('status', { name });
      expect(chip).toHaveClass('rounded-selector', 'bg-base-300/60');
      expect(chip.className).not.toMatch(/error|danger|success|info/);
    }
  });
});
