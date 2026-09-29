import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

// vi.mock factories are hoisted above all imports, so the mocked
// module is in place before the SUT loads useT from @/i18n.
vi.mock('@/i18n', () => ({
  useT: () => ({
    t: (key: string) => {
      // Minimal stub: anything that starts with "configuration:topology."
      // returns a readable tag so the diagram renders its labels.
      const tail = key.split('.').pop() ?? key;
      return `topology.${tail}`;
    },
  }),
}));

import { NrccTopologyDiagram } from './NrccTopologyDiagram';

describe('NrccTopologyDiagram', () => {
  it('renders the figure with the default topology title', () => {
    render(<NrccTopologyDiagram />);
    expect(
      screen.getByRole('img', { name: 'topology.title' })
    ).toBeInTheDocument();
  });

  it('honours an explicit ariaLabel override', () => {
    render(<NrccTopologyDiagram ariaLabel="Custom diagram label" />);
    expect(
      screen.getByRole('img', { name: 'Custom diagram label' })
    ).toBeInTheDocument();
  });

  it('renders three node labels: NRCC, settings.js, Node-RED', () => {
    render(<NrccTopologyDiagram />);
    // text nodes inside the SVG carry the node identifiers.
    expect(screen.getByText('NRCC')).toBeInTheDocument();
    expect(screen.getByText('settings.js')).toBeInTheDocument();
    expect(screen.getByText('Node-RED')).toBeInTheDocument();
  });

  it('renders the topology subtitle as accessible description text', () => {
    render(<NrccTopologyDiagram />);
    expect(screen.getByText('topology.subtitle')).toBeInTheDocument();
  });
});
