import { AlertTriangle, CheckCircle2, Link as LinkIcon } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { HostStatus } from '@/shared/types';
import { getSystemHealthIssues } from '../lib';

interface SystemHealthCardProps {
  host?: HostStatus;
}

export function SystemHealthCard({ host }: SystemHealthCardProps) {
  if (!host) {
    return null;
  }

  const issues = getSystemHealthIssues(host);

  return (
    <div className="card surface-card border border-border p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {host.ready ? (
            <CheckCircle2 className="h-5 w-5 text-success" />
          ) : (
            <AlertTriangle className="h-5 w-5 text-warning" />
          )}
          <div>
            <span className="text-sm font-medium">System Health</span>
            <p className="mt-0.5 text-xs text-body-secondary">
              {host.ready ? 'Environment OK' : 'Check environment for issues'}
            </p>
          </div>
        </div>
        <Link to="/bootstrap" className="btn btn-ghost btn-sm gap-1 text-xs">
          <LinkIcon className="h-3.5 w-3.5" />
          View Details
        </Link>
      </div>

      {!host.ready && (
        <div className="mt-3 space-y-1 text-xs">
          {issues.map((issue) => (
            <p key={issue} className="text-warning">
              • {issue}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
