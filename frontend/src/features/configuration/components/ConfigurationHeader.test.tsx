import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

vi.mock('@/i18n', () => ({
  useT: () => ({
    t: (key: string) => key,
  }),
}));

import { ConfigurationHeader } from './ConfigurationHeader';

describe('ConfigurationHeader', () => {
  it('renders the page title and a Settings icon', () => {
    render(<ConfigurationHeader />);
    expect(
      screen.getByRole('heading', { name: 'configuration:pageTitle' })
    ).toBeInTheDocument();
  });

  it('renders the topology diagram by default', () => {
    render(<ConfigurationHeader />);
    expect(screen.getByTestId('nrcc-topology-diagram')).toBeInTheDocument();
  });

  it('hides the topology diagram when hideTopology is true', () => {
    render(<ConfigurationHeader hideTopology />);
    expect(screen.queryByTestId('nrcc-topology-diagram')).toBeNull();
  });
});
