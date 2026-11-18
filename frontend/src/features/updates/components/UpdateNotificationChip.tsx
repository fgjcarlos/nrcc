import { ArrowUpCircle, X, Loader2, CheckCircle2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useUpdateStatus } from '@/features/updates/hooks';
import { useUpdateFlowState } from '@/features/updates/hooks';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { useState, useEffect } from 'react';
import { useT } from '@/i18n';

const DISMISS_KEY = 'cc-update-dismissed-version';

export function UpdateNotificationChip() {
  const navigate = useNavigate();
  const { data: status } = useUpdateStatus();
  const { data: flowState } = useUpdateFlowState();
  const { user } = useAuth();
  const [dismissed, setDismissed] = useState<string | null>(null);
  const { t } = useT();

  // Load dismissed version from localStorage on mount
  useEffect(() => {
    const dismissedVersion = localStorage.getItem(DISMISS_KEY);
    setDismissed(dismissedVersion);
  }, []);

  // Determine which badge to show
  const isUpdateActive = flowState?.state && ['BackingUp', 'Applying'].includes(flowState.state);
  const isUpdateCompleted = flowState?.state === 'Completed';
  const hasAvailableUpdate = status?.updateAvailable && !status?.error;
  const isAdmin = user?.role === 'admin';
  
  // Show update available badge if there's an update and it hasn't been dismissed
  // (dismissed version is different from latest version)
  if (isAdmin && hasAvailableUpdate && dismissed !== status?.latestVersion && !isUpdateActive) {
    const handleDismiss = (e: React.MouseEvent) => {
      e.stopPropagation();
      localStorage.setItem(DISMISS_KEY, status.latestVersion);
      setDismissed(status.latestVersion);
    };

    const handleClick = () => {
      navigate('/maintenance/updates');
    };

    return (
      <div
        className="flex h-11 items-stretch text-ds-on-tint-primary"
        role="alert"
        aria-label={t('updates:updateAvailable')}
      >
        <button
          onClick={handleClick}
          className="flex min-w-11 flex-1 items-center justify-center gap-2 rounded-l-field border border-primary bg-ds-bg-surface px-3 transition-colors hover:bg-ds-accent-dim"
          aria-label={t('updates:availableAria')}
        >
          <ArrowUpCircle className="h-4 w-4 shrink-0" aria-hidden="true" />
          {/* Icon-only below xl so the header controls never push past the viewport. */}
          <span className="sr-only text-sm font-medium xl:not-sr-only">{t('updates:updateAvailable')}</span>
        </button>
        <button
          onClick={handleDismiss}
          className="grid w-11 shrink-0 place-items-center rounded-r-field border border-l-0 border-primary bg-ds-bg-surface text-ds-text-secondary transition-colors hover:bg-base-200 hover:text-ds-text-primary"
          aria-label={t('updates:dismiss')}
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    );
  }

  // Show update in progress badge
  if (isUpdateActive) {
    return (
      <div
        className="flex h-11 items-center gap-2 rounded-field border border-ds-border-strong bg-ds-bg-surface px-3 text-ds-on-tint-info"
        role="status"
        aria-live="polite"
        aria-label={t('updates:inProgress')}
      >
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" aria-hidden="true" />
        <span className="sr-only text-sm font-medium xl:not-sr-only">
          {flowState?.state === 'BackingUp' ? t('updates:backingUp') : t('updates:updating')}
        </span>
      </div>
    );
  }

  // Show update completed badge (persistent for user awareness)
  if (isUpdateCompleted) {
    return (
      <div
        className="flex h-11 items-center gap-2 rounded-field border border-ds-border-strong bg-ds-bg-surface px-3 text-ds-on-tint-success"
        role="status"
        aria-label={t('updates:completed')}
      >
        <CheckCircle2 className="h-4 w-4 shrink-0" aria-hidden="true" />
        <span className="sr-only text-sm font-medium xl:not-sr-only">{t('updates:upToDate')}</span>
      </div>
    );
  }

  // No badge when up to date and no active operation
  return null;
}
