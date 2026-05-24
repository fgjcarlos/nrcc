import {
  AreaChart,
  Area,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { MetricsSnapshot } from '../types/history';

interface MetricsChartProps {
  data: MetricsSnapshot[];
  dataKey: keyof Omit<MetricsSnapshot, 'timestamp'>;
  label: string;
  color: string;
  loading?: boolean;
}

function formatTime(timestamp: string): string {
  const d = new Date(timestamp);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export function MetricsChart({ data, dataKey, label, color, loading = false }: MetricsChartProps) {
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
          <span className="text-xs text-body-secondary">No data yet</span>
        </div>
      </div>
    );
  }

  const chartData = data.map((snapshot) => ({
    time: formatTime(snapshot.timestamp),
    value: snapshot[dataKey],
  }));

  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium text-body-secondary">{label}</span>
      <ResponsiveContainer width="100%" height={64}>
        <AreaChart data={chartData} margin={{ top: 0, right: 0, left: -24, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(128,128,128,0.15)" />
          <XAxis
            dataKey="time"
            tick={{ fontSize: 9 }}
            tickLine={false}
            axisLine={false}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={[0, 100]}
            tick={{ fontSize: 9 }}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip
            contentStyle={{ fontSize: 11 }}
            formatter={(value) => {
              const num = typeof value === 'number' ? value : Number(value);
              return [`${num.toFixed(1)}%`, label];
            }}
            labelFormatter={(l) => `Time: ${l}`}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={color}
            fillOpacity={0.15}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
