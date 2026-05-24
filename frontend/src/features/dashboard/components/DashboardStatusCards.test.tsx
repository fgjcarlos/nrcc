import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DashboardStatusCards } from './DashboardStatusCards';
import type { SystemInfo } from '@/shared/types';
import type { MetricsSnapshot } from '../types/history';

// Stub recharts to avoid jsdom incompatibilities
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div data-testid="area-chart">{children}</div>,
  Area: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
}));

// Stub useSystemHistory hook
vi.mock('../hooks/useSystemHistory', () => ({
  useSystemHistory: vi.fn(),
}));

import * as useSystemHistoryModule from '../hooks/useSystemHistory';

const mockSystem: SystemInfo = {
  cpu: { usage: 42.5, cores: 4 },
  memory: { total: 8000000000, used: 4000000000, free: 4000000000, usagePercent: 50 },
  disk: { total: 100000000000, used: 60000000000, free: 40000000000, usagePercent: 60 },
  uptime: 3600,
  platform: 'linux',
  hostname: 'server',
};

const mockHistory: MetricsSnapshot[] = [
  { timestamp: '2024-01-01T00:00:00Z', cpuPercent: 30, memoryPercent: 45, diskPercent: 58 },
  { timestamp: '2024-01-01T00:00:30Z', cpuPercent: 42.5, memoryPercent: 50, diskPercent: 60 },
];

describe('DashboardStatusCards — with system history', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('displays CPU percentage value and chart when system and history data are available', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    render(
      <DashboardStatusCards
        system={mockSystem}
        inDocker={false}
        container={null}
        host={undefined}
      />
    );

    // formatPercent rounds: 42.5 -> 43%
    expect(screen.getByText('43%')).toBeInTheDocument();
    expect(screen.getAllByTestId('area-chart').length).toBeGreaterThan(0);
  });

  it('displays memory percentage value when system data is available', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    render(
      <DashboardStatusCards
        system={mockSystem}
        inDocker={false}
        container={null}
        host={undefined}
      />
    );

    expect(screen.getByText('50%')).toBeInTheDocument();
  });

  it('displays Disk usage chart section', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    render(
      <DashboardStatusCards
        system={mockSystem}
        inDocker={false}
        container={null}
        host={undefined}
      />
    );

    expect(screen.getByText('Disk')).toBeInTheDocument();
    expect(screen.getByText('60%')).toBeInTheDocument();
  });

  it('shows chart loading skeletons when history is loading', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });

    render(
      <DashboardStatusCards
        system={mockSystem}
        inDocker={false}
        container={null}
        host={undefined}
      />
    );

    // Loading skeletons have role="status"
    const skeletons = screen.getAllByRole('status');
    expect(skeletons.length).toBeGreaterThanOrEqual(3);
  });
});
