import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { classifyPosture, SecurityPostureCard } from './SecurityPostureCard';

const healthy = {
  encryptionKeyConfigured: true,
  backupDownloadAdminOnly: true,
  activeRefreshSessions: 0,
  mfa: { enrolledAdmins: 2, totalAdmins: 2 },
};

describe('SecurityPostureCard', () => {
  it('renders healthy encryption and backup states from canonical data', () => {
    render(<SecurityPostureCard posture={healthy} isLoading={false} isError={false} />);

    expect(screen.getByRole('status', { name: /Encryption key: configured, healthy/i })).toBeInTheDocument();
    expect(screen.getByRole('status', { name: /Backup access: admin-only, healthy/i })).toBeInTheDocument();
    expect(screen.getByTestId('security-posture-card')).toHaveAttribute('data-severity', 'success');
  });

  it('makes missing encryption and non-admin backup critical with a plaintext warning', () => {
    render(<SecurityPostureCard posture={{ ...healthy, encryptionKeyConfigured: false, backupDownloadAdminOnly: false }} isLoading={false} isError={false} />);

    expect(screen.getByRole('status', { name: /Encryption key: not configured, critical/i })).toBeInTheDocument();
    expect(screen.getByText('encrypted variables are silently stored in plaintext without `NRCC_ENCRYPTION_KEY`')).toBeInTheDocument();
    expect(screen.getByRole('status', { name: /Backup access: not admin-only, critical/i })).toBeInTheDocument();
    expect(screen.getByTestId('security-posture-card')).toHaveAttribute('data-severity', 'error');
  });

  it('classifies sessions as healthy, degraded, and critical when unavailable', () => {
    const { rerender } = render(<SecurityPostureCard posture={healthy} isLoading={false} isError={false} />);
    expect(screen.getByRole('status', { name: /Sessions: 0 active, healthy/i })).toBeInTheDocument();

    rerender(<SecurityPostureCard posture={{ ...healthy, activeRefreshSessions: 3 }} isLoading={false} isError={false} />);
    expect(screen.getByRole('status', { name: /Sessions: 3 active, degraded/i })).toBeInTheDocument();

    rerender(<SecurityPostureCard posture={undefined} isLoading={false} isError={true} />);
    expect(screen.getByRole('status', { name: /Sessions: unavailable, critical/i })).toBeInTheDocument();
  });

  it('classifies MFA enrollment across healthy, degraded, and critical states', () => {
    const { rerender } = render(<SecurityPostureCard posture={healthy} isLoading={false} isError={false} />);
    expect(screen.getByRole('status', { name: /MFA: 2 of 2 enrolled, healthy/i })).toBeInTheDocument();

    rerender(<SecurityPostureCard posture={{ ...healthy, mfa: { enrolledAdmins: 1, totalAdmins: 2 } }} isLoading={false} isError={false} />);
    expect(screen.getByRole('status', { name: /MFA: 1 of 2 enrolled, degraded/i })).toBeInTheDocument();

    rerender(<SecurityPostureCard posture={{ ...healthy, mfa: { enrolledAdmins: 0, totalAdmins: 0 } }} isLoading={false} isError={false} />);
    expect(screen.getByRole('status', { name: /MFA: 0 of 0 enrolled, critical/i })).toBeInTheDocument();
  });

  it('makes every posture chip focusable with its name, value, and state', () => {
    render(<SecurityPostureCard posture={healthy} isLoading={false} isError={false} />);

    expect(screen.getAllByRole('status')).toHaveLength(4);
    for (const chip of screen.getAllByRole('status')) {
      expect(chip).toHaveAttribute('tabindex', '0');
      expect(chip).toHaveAccessibleName(/.+: .+, (healthy|degraded|critical)/i);
    }
  });
  it('centralizes all four unavailable classifications', () => {
    expect(classifyPosture(undefined)).toEqual([
      { name: 'Encryption key', value: 'unavailable', state: 'critical', severity: 'error' },
      { name: 'Backup access', value: 'unavailable', state: 'critical', severity: 'error' },
      { name: 'Sessions', value: 'unavailable', state: 'critical', severity: 'error' },
      { name: 'MFA', value: 'unavailable', state: 'critical', severity: 'error' },
    ]);
  });

});
