import { Activity, LucideIcon } from 'lucide-react';

interface MetricCardProps {
  icon: LucideIcon;
  label: string;
  value: number;
  warning?: boolean;
}

export function MetricCard({ icon: Icon, label, value, warning }: MetricCardProps) {
  return (
    <div className="surface-card p-4">
      <div className="mb-1 flex items-center gap-2 text-base-content/60">
        <Icon className="w-4 h-4" />
        <span className="text-sm">{label}</span>
      </div>
      <p className={`text-2xl font-bold ${warning ? 'text-amber-300' : 'text-base-content'}`}>
        {value}
      </p>
    </div>
  );
}
