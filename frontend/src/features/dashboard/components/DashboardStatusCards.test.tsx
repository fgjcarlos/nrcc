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

const noop = () => {};

function renderCards(overrides: Partial<React.ComponentProps<typeof DashboardStatusCards>> = {}) {
  return render(
    <DashboardStatusCards
      system={mockSystem}
      inDocker={false}
      container={null}
      host={undefined}
      isRestarting={false}
      onRequestRestart={noop}
      onOpenNodeRed={noop}
      {...overrides}
    />
  );
}

describe('DashboardStatusCards — Runtime + metric charts', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('displays CPU percentage value and chart when system and history data are available', async () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    renderCards();

    // formatPercent rounds: 42.5 -> 43%
    expect(screen.getByText('43%')).toBeInTheDocument();
    // MetricsChart is lazy-loaded (#301), so the chart appears after the chunk resolves.
    expect((await screen.findAllByTestId('area-chart')).length).toBeGreaterThan(0);
  });

  it('displays memory percentage value when system data is available', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    renderCards();

    expect(screen.getByText('50%')).toBeInTheDocument();
  });

  it('does NOT render a Disk chart — Disk moved out of the top row (issue #676 item 3)', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    renderCards();

    // No "Disk" header in the metric row. The disk card now lives in
    // DashboardDetails below.
    expect(screen.queryByText('Disk')).not.toBeInTheDocument();
  });

  it('promotes the Runtime card with status and image, plus Restart + Open actions', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    renderCards({ inDocker: true, container: { inDocker: true, status: 'running', image: 'nodered:4.0.0' } });

    expect(screen.getByText('Runtime')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
    expect(screen.getByText('nodered:4.0.0')).toBeInTheDocument();
    // Spanish copy kept in sync with the existing dashboard (issue #676
    // explicitly defers the Spanish→English cleanup to #677 follow-ups).
    expect(screen.getByRole('button', { name: 'Reiniciar' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Abrir' })).toBeInTheDocument();
  });

  it('shows Restart button as "Reiniciando…" and disables it while isRestarting', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: mockHistory,
      isLoading: false,
      isError: false,
    });

    renderCards({ isRestarting: true });

    const restartBtn = screen.getByRole('button', { name: 'Reiniciando…' });
    expect(restartBtn).toBeInTheDocument();
    expect(restartBtn).toBeDisabled();
  });

  it('shows chart loading skeletons when history is loading', () => {
    vi.mocked(useSystemHistoryModule.useSystemHistory).mockReturnValue({
      data: [],
      isLoading: true,
      isError: false,
    });

    renderCards();

    // Two charts in the top row now: CPU + Memory (Disk moved out).
    const skeletons = screen.getAllByRole('status');
    expect(skeletons.length).toBe(2);
  });
});
