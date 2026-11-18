import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MetricsChart } from './MetricsChart';
import type { MetricsSnapshot } from '../types/history';

// recharts uses browser APIs not available in jsdom; stub the whole module
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div data-testid="area-chart">{children}</div>,
  Area: ({ stroke, isAnimationActive }: { stroke?: string; isAnimationActive?: boolean }) => (
    <div data-testid="area" data-stroke={stroke} data-animated={String(isAnimationActive)} />
  ),
  XAxis: () => <div data-testid="xaxis" />,
  YAxis: () => <div data-testid="yaxis" />,
  CartesianGrid: ({ stroke, strokeDasharray }: { stroke?: string; strokeDasharray?: string }) => (
    <div data-testid="cartesian-grid" data-stroke={stroke} data-dash={strokeDasharray ?? ''} />
  ),
  Tooltip: ({ isAnimationActive }: { isAnimationActive?: boolean }) => (
    <div data-testid="tooltip" data-animated={String(isAnimationActive)} />
  ),
}));

const mockData: MetricsSnapshot[] = [
  { timestamp: '2024-01-01T00:00:00Z', cpuPercent: 30, memoryPercent: 50, diskPercent: 70 },
  { timestamp: '2024-01-01T00:00:30Z', cpuPercent: 35, memoryPercent: 55, diskPercent: 72 },
];

describe('MetricsChart — loading state', () => {
  it('shows a skeleton placeholder when loading is true', () => {
    render(<MetricsChart data={[]} dataKey="cpuPercent" label="CPU" loading />);

    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.queryByTestId('area-chart')).not.toBeInTheDocument();
  });
});

describe('MetricsChart — empty state', () => {
  it('shows "No data yet" message when data array is empty and not loading', () => {
    render(<MetricsChart data={[]} dataKey="cpuPercent" label="CPU" loading={false} />);

    expect(screen.getByText('No data yet')).toBeInTheDocument();
    expect(screen.queryByTestId('area-chart')).not.toBeInTheDocument();
  });
});

describe('MetricsChart — data state', () => {
  it('renders a responsive area chart when data is present', () => {
    render(
      <MetricsChart
        data={mockData}
        dataKey="cpuPercent"
        label="CPU"
        loading={false}
      />
    );

    expect(screen.getByTestId('area-chart')).toBeInTheDocument();
    expect(screen.queryByText('No data yet')).not.toBeInTheDocument();
  });

  it('shows the label above the chart', () => {
    render(
      <MetricsChart
        data={mockData}
        dataKey="memoryPercent"
        label="Memory"
        loading={false}
      />
    );

    expect(screen.getByText('Memory')).toBeInTheDocument();
  });
});

// T8: token colours read at runtime, hairline grid, accessible summary, no motion.
describe('MetricsChart — tokens and accessibility', () => {
  afterEach(() => {
    document.documentElement.style.removeProperty('--ds-border-subtle');
    document.documentElement.style.removeProperty('--ds-text-secondary');
  });

  it('draws the grid and series from CSS custom properties read at runtime', () => {
    document.documentElement.style.setProperty('--ds-border-subtle', '#d6d4cb');
    document.documentElement.style.setProperty('--ds-text-secondary', '#3d424b');
    render(<MetricsChart data={mockData} dataKey="cpuPercent" label="CPU" />);

    const grid = screen.getByTestId('cartesian-grid');
    expect(grid).toHaveAttribute('data-stroke', '#d6d4cb');
    expect(grid).toHaveAttribute('data-dash', '');
    expect(screen.getByTestId('area')).toHaveAttribute('data-stroke', '#3d424b');
  });

  it('exposes an accessible summary of the series', () => {
    render(<MetricsChart data={mockData} dataKey="cpuPercent" label="CPU" />);

    expect(screen.getByRole('img', { name: 'CPU: 2 samples, latest 35.0%, range 30.0–35.0%' })).toBeInTheDocument();
  });

  it('does not animate the series or the tooltip', () => {
    render(<MetricsChart data={mockData} dataKey="cpuPercent" label="CPU" />);

    expect(screen.getByTestId('area')).toHaveAttribute('data-animated', 'false');
    expect(screen.getByTestId('tooltip')).toHaveAttribute('data-animated', 'false');
  });
});
