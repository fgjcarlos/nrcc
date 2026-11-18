import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

vi.mock('@/i18n', () => ({
  useT: () => ({
    t: (key: string) => key,
  }),
}));

import { ConfigurationHeader } from './ConfigurationHeader';

describe('ConfigurationHeader', () => {
  it('renders the page title with its designator and status', () => {
    render(<ConfigurationHeader designator={<span>REV 9f2c1b0d</span>} status={<p>Node-RED 5.0.6</p>} />);
    expect(
      screen.getByRole('heading', { name: 'configuration:pageTitle' })
    ).toBeInTheDocument();
    expect(screen.getByText('REV 9f2c1b0d')).toBeInTheDocument();
    expect(screen.getByText('Node-RED 5.0.6')).toBeInTheDocument();
  });

  it('no longer embeds the topology diagram', () => {
    render(<ConfigurationHeader />);
    expect(screen.queryByTestId('nrcc-topology-diagram')).toBeNull();
  });
});
