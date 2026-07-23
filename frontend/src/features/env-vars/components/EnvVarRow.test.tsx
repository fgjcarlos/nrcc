import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { EnvVarRow } from './EnvVarRow';
import type { EnvVar } from '../services/envService';

function renderRow(envVar: EnvVar) {
  return render(
    <table>
      <tbody>
        <EnvVarRow
          envVar={envVar}
          onDelete={vi.fn()}
          onEdit={vi.fn()}
          onToggleSecret={vi.fn()}
          showSecret={false}
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
});
