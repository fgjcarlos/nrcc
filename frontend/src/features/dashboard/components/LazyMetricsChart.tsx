import { lazy, Suspense } from 'react';
import type { MetricsSnapshot } from '../types/history';

// recharts is ~300 kB and was being pulled into the DashboardView route chunk
// (the landing route after login). Loading MetricsChart lazily moves recharts
// into a deferred chunk so the dashboard shell paints without it. (#301)
const MetricsChart = lazy(() =>
  import('./MetricsChart').then((m) => ({ default: m.MetricsChart }))
);

interface LazyMetricsChartProps {
  data: MetricsSnapshot[];
  dataKey: keyof Omit<MetricsSnapshot, 'timestamp'>;
  label: string;
  color: string;
  loading?: boolean;
}

export function LazyMetricsChart(props: LazyMetricsChartProps) {
  // Fallback mirrors MetricsChart's own loading skeleton (label + h-16) so there
  // is no layout shift while the recharts chunk downloads.
  return (
    <Suspense
      fallback={
        <div className="flex flex-col gap-1">
          <span className="text-xs font-medium text-body-secondary">{props.label}</span>
          <div
            role="status"
            className="h-16 animate-pulse rounded bg-base-300"
            aria-label="Loading chart"
          />
        </div>
      }
    >
      <MetricsChart {...props} />
    </Suspense>
  );
}
