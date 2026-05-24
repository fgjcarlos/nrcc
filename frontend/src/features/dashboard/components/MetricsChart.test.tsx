import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MetricsChart } from './MetricsChart';
import type { MetricsSnapshot } from '../types/history';

// recharts uses browser APIs not available in jsdom; stub the whole module
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div data-testid="area-chart">{children}</div>,
  Area: () => <div data-testid="area" />,
  XAxis: () => <div data-testid="xaxis" />,
  YAxis: () => <div data-testid="yaxis" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  Tooltip: () => <div data-testid="tooltip" />,
}));

const mockData: MetricsSnapshot[] = [
  { timestamp: '2024-01-01T00:00:00Z', cpuPercent: 30, memoryPercent: 50, diskPercent: 70 },
  { timestamp: '2024-01-01T00:00:30Z', cpuPercent: 35, memoryPercent: 55, diskPercent: 72 },
];

describe('MetricsChart — loading state', () => {
  it('shows a skeleton placeholder when loading is true', () => {
    render(<MetricsChart data={[]} dataKey="cpuPercent" label="CPU" color="#3b82f6" loading />);

    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.queryByTestId('area-chart')).not.toBeInTheDocument();
  });
});

describe('MetricsChart — empty state', () => {
  it('shows "No data yet" message when data array is empty and not loading', () => {
    render(<MetricsChart data={[]} dataKey="cpuPercent" label="CPU" color="#3b82f6" loading={false} />);

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
        color="#3b82f6"
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
        color="#8b5cf6"
        loading={false}
      />
    );

    expect(screen.getByText('Memory')).toBeInTheDocument();
  });
});
