import { useEffect, useState } from 'react';
import {
  AreaChart,
  Area,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import { useT } from '@/i18n';
import type { MetricsSnapshot } from '../types/history';

interface MetricsChartProps {
  data: MetricsSnapshot[];
  dataKey: keyof Omit<MetricsSnapshot, 'timestamp'>;
  label: string;
  /** CSS custom property for the series stroke; defaults to secondary ink. */
  colorVar?: `--${string}`;
  loading?: boolean;
}

function formatTime(timestamp: string): string {
  const d = new Date(timestamp);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function readColors(seriesVar: string) {
  const style = getComputedStyle(document.documentElement);
  const read = (name: string) => style.getPropertyValue(name).trim();
  return { grid: read('--ds-border-subtle'), axis: read('--ds-text-muted'), series: read(seriesVar) };
}

/** Token colours read at runtime; re-read when the theme attribute flips. */
function useTokenColors(seriesVar: string) {
  const [colors, setColors] = useState(() => readColors(seriesVar));
  useEffect(() => {
    const observer = new MutationObserver(() => setColors(readColors(seriesVar)));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    return () => observer.disconnect();
  }, [seriesVar]);
  return colors;
}

export function MetricsChart({ data, dataKey, label, colorVar = '--ds-text-secondary', loading = false }: MetricsChartProps) {
  const { t } = useT();
  const colors = useTokenColors(colorVar);
  if (loading) {
    return (
      <div className="flex flex-col gap-1">
        <span className="text-xs font-medium text-body-secondary">{label}</span>
        <div role="status" className="h-16 animate-pulse rounded bg-base-300" aria-label="Loading chart" />
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="flex flex-col gap-1">
        <span className="text-xs font-medium text-body-secondary">{label}</span>
        <div className="flex h-16 items-center justify-center rounded border border-border bg-base-200">
          <span className="text-xs text-body-secondary">{t('common:noDataYet')}</span>
        </div>
      </div>
    );
  }

  const chartData = data.map((snapshot) => ({
    time: formatTime(snapshot.timestamp),
    value: snapshot[dataKey],
  }));
  const values = chartData.map((d) => d.value);
  const summary = t('dashboard:chart.summary', {
    label,
    count: values.length,
    latest: values[values.length - 1].toFixed(1),
    min: Math.min(...values).toFixed(1),
    max: Math.max(...values).toFixed(1),
  });
  const tick = { fontSize: 12, fill: colors.axis };

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium text-body-secondary">{label}</span>
      {/* The summary names the figure; the SVG itself stays out of the tab order. */}
      <div role="img" aria-label={summary}>
        <ResponsiveContainer width="100%" height={64}>
          <AreaChart data={chartData} margin={{ top: 0, right: 0, left: -24, bottom: 0 }} accessibilityLayer={false}>
            <CartesianGrid stroke={colors.grid} strokeWidth={1} vertical={false} />
            <XAxis
              dataKey="time"
              tick={tick}
              tickLine={false}
              axisLine={false}
              interval="preserveStartEnd"
            />
            <YAxis
              domain={[0, 100]}
              tick={tick}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip
              isAnimationActive={false}
              contentStyle={{ fontSize: 12 }}
              formatter={(value) => {
                const num = typeof value === 'number' ? value : Number(value);
                return [`${num.toFixed(1)}%`, label];
              }}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke={colors.series}
              fill={colors.series}
              fillOpacity={0.15}
              strokeWidth={1.5}
              dot={false}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
