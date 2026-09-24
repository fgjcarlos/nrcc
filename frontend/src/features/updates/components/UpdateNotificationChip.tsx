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
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-success/10 text-success"
        role="alert"
        aria-label={t('updates:updateAvailable')}
      >
        <button
          onClick={handleClick}
          className="flex items-center gap-2 flex-1 hover:opacity-80 transition-opacity"
          aria-label={t('updates:availableAria')}
        >
          <ArrowUpCircle className="w-4 h-4 flex-shrink-0" />
          <span className="text-sm font-medium">{t('updates:updateAvailable')}</span>
        </button>
        <button
          onClick={handleDismiss}
          className="ml-2 p-0.5 rounded hover:bg-success/20 opacity-60 hover:opacity-100 transition-all flex-shrink-0"
          aria-label={t('updates:dismiss')}
        >
          <X className="w-3 h-3" />
        </button>
      </div>
    );
  }

  // Show update in progress badge
  if (isUpdateActive) {
    return (
      <div
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-accent/10 text-accent"
        role="status"
        aria-live="polite"
        aria-label={t('updates:inProgress')}
      >
        <Loader2 className="w-4 h-4 animate-spin flex-shrink-0" />
        <span className="text-sm font-medium">
          {flowState?.state === 'BackingUp' ? t('updates:backingUp') : t('updates:updating')}
        </span>
      </div>
    );
  }

  // Show update completed badge (persistent for user awareness)
  if (isUpdateCompleted) {
    return (
      <div
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-success/10 text-success"
        role="status"
        aria-label={t('updates:completed')}
      >
        <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
        <span className="text-sm font-medium">{t('updates:upToDate')}</span>
      </div>
    );
  }

  // No badge when up to date and no active operation
  return null;
}
