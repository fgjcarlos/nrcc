import { CheckCircle2, CircleAlert, TriangleAlert } from 'lucide-react';
import type { SecurityPosture } from '../types';

type Severity = 'success' | 'warning' | 'error';

interface PostureChip {
  name: string;
  value: string;
  state: 'healthy' | 'degraded' | 'critical';
  severity: Severity;
}

export interface SecurityPostureCardProps {
  posture?: SecurityPosture;
  isLoading: boolean;
  isError: boolean;
}

function unavailableChip(name: string): PostureChip {
  return { name, value: 'unavailable', state: 'critical', severity: 'error' };
}

export function classifyPosture(posture?: SecurityPosture): PostureChip[] {
  if (!posture) return ['Encryption key', 'Backup access', 'Sessions', 'MFA'].map(unavailableChip);

  const encryption = posture.encryptionKeyConfigured
    ? { value: 'configured', state: 'healthy' as const, severity: 'success' as const }
    : { value: 'not configured', state: 'critical' as const, severity: 'error' as const };
  const backup = posture.backupDownloadAdminOnly
    ? { value: 'admin-only', state: 'healthy' as const, severity: 'success' as const }
    : { value: 'not admin-only', state: 'critical' as const, severity: 'error' as const };
  const sessions = posture.activeRefreshSessions === 0
    ? { value: '0 active', state: 'healthy' as const, severity: 'success' as const }
    : { value: `${posture.activeRefreshSessions} active`, state: 'degraded' as const, severity: 'warning' as const };
  const { enrolledAdmins, totalAdmins } = posture.mfa;
  const mfa = totalAdmins > 0 && enrolledAdmins === totalAdmins
    ? { value: `${enrolledAdmins} of ${totalAdmins} enrolled`, state: 'healthy' as const, severity: 'success' as const }
    : totalAdmins > 0 && enrolledAdmins > 0
      ? { value: `${enrolledAdmins} of ${totalAdmins} enrolled`, state: 'degraded' as const, severity: 'warning' as const }
      : { value: `${enrolledAdmins} of ${totalAdmins} enrolled`, state: 'critical' as const, severity: 'error' as const };

  return [
    { name: 'Encryption key', ...encryption },
    { name: 'Backup access', ...backup },
    { name: 'Sessions', ...sessions },
    { name: 'MFA', ...mfa },
  ];
}

const severityRank: Record<Severity, number> = { success: 0, warning: 1, error: 2 };
const severityIcon = { success: CheckCircle2, warning: TriangleAlert, error: CircleAlert };

export function SecurityPostureCard({ posture, isLoading, isError }: SecurityPostureCardProps) {
  const chips = classifyPosture(posture);
  const severity = chips.reduce<Severity>((current, chip) => severityRank[chip.severity] > severityRank[current] ? chip.severity : current, 'success');

  return (
    <section data-testid="security-posture-card" data-severity={severity} className="p-6 border card surface-card border-border" aria-label="Security posture">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold">Security posture</h2>
        <span className="text-sm capitalize text-body-secondary">{isLoading ? 'Loading' : isError ? 'Unavailable' : severity === 'success' ? 'Healthy' : severity === 'warning' ? 'Degraded' : 'Critical'}</span>
      </div>
      <div className="grid grid-cols-1 gap-3 mt-4 sm:grid-cols-2">
        {chips.map((chip) => {
          const Icon = severityIcon[chip.severity];
          return <div key={chip.name} role="status" tabIndex={0} aria-label={`${chip.name}: ${chip.value}, ${chip.state}`} className="rounded-xl border border-border p-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary">
            <Icon aria-hidden="true" className="inline w-4 h-4 mr-2" />
            <span className="font-medium">{chip.name}</span>: {chip.value}, <span className="capitalize">{chip.state}</span>
          </div>;
        })}
      </div>
      {!posture?.encryptionKeyConfigured && <p role="alert" className="mt-4 text-sm text-error">encrypted variables are silently stored in plaintext without `NRCC_ENCRYPTION_KEY`</p>}
    </section>
  );
}
