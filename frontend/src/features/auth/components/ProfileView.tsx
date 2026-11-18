import { FormEvent, Fragment, useState } from 'react';
import { toast } from 'sonner';
import { useAuth } from '../hooks/useAuth';
import { authService } from '../services/authService';
import { useT } from '@/i18n';
import { Button, Input, StatusChip } from '@/shared/components/ui';

const MIN_PASSWORD_LENGTH = 8;

function formatDate(value: string | undefined, fallback: string) {
  if (!value) return fallback;

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return fallback;

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

/** Designator text that may wrap only after a hyphen (DESIGN.md: never mid-word). */
function BreakAtHyphens({ value }: { value: string }) {
  const parts = value.split('-');
  return parts.map((part, index) => (
    <Fragment key={index}>
      {part}
      {index < parts.length - 1 && <>{'-'}<wbr /></>}
    </Fragment>
  ));
}

export function ProfileView() {
  const { user } = useAuth();
  const { t } = useT();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!user) {
    return (
      <div className="space-y-1">
        <h1 className="text-xl font-bold text-ds-text-primary">{t('auth:profile.title')}</h1>
        <p className="text-sm text-ds-text-secondary">{t('auth:profile.signInRequired')}</p>
      </div>
    );
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(t('auth:setup.errors.passwordLength'));
      return;
    }

    if (password !== confirmPassword) {
      setError(t('auth:setup.errors.passwordMismatch'));
      return;
    }

    setIsSubmitting(true);
    try {
      await authService.changePassword(user.id, password);
      setPassword('');
      setConfirmPassword('');
      toast.success(t('auth:profile.passwordUpdated'));
    } catch (err) {
      const message =
        (err as { response?: { data?: { error?: { message?: string } } } }).response?.data?.error?.message ||
        t('auth:profile.updateFailed');
      setError(message);
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const notAvailable = t('auth:profile.notAvailable');

  return (
    <div className="space-y-4">
      <header className="space-y-1 border-b border-ds-border-subtle pb-3">
        <h1 className="text-xl font-bold text-ds-text-primary">{t('auth:profile.title')}</h1>
        <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('auth:profile.reviewPrompt')}</p>
      </header>

      <div className="divide-y divide-ds-border-subtle">
        <section aria-labelledby="profile-account-heading" className="space-y-4 pb-4">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <h2 id="profile-account-heading" className="min-w-0 text-sm font-semibold text-ds-text-primary [overflow-wrap:anywhere]">
              {user.username}
            </h2>
            <StatusChip variant="neutral" size="sm">
              {user.role === 'admin' ? t('auth:roles.admin') : t('auth:roles.viewer')}
            </StatusChip>
          </div>

          <dl className="grid gap-4 md:grid-cols-3">
            <div className="min-w-0 space-y-1">
              <dt className="text-xs text-ds-text-secondary">{t('auth:profile.userId')}</dt>
              <dd className="font-mono text-xs font-semibold tabular-nums text-ds-text-primary break-normal">
                <BreakAtHyphens value={user.id} />
              </dd>
            </div>
            <div className="min-w-0 space-y-1">
              <dt className="text-xs text-ds-text-secondary">{t('auth:createdLabel')}</dt>
              <dd className="text-sm text-ds-text-primary">{formatDate(user.createdAt, notAvailable)}</dd>
            </div>
            <div className="min-w-0 space-y-1">
              <dt className="text-xs text-ds-text-secondary">{t('auth:profile.lastUpdated')}</dt>
              <dd className="text-sm text-ds-text-primary">{formatDate(user.updatedAt, notAvailable)}</dd>
            </div>
          </dl>
        </section>

        <section aria-labelledby="profile-password-heading" className="space-y-3 pt-4">
          <h2 id="profile-password-heading" className="text-sm font-semibold text-ds-text-primary">
            {t('auth:profile.passwordHeading')}
          </h2>

          {/* noValidate: the same length and match checks run in handleSubmit and answer inline. */}
          <form onSubmit={handleSubmit} noValidate className="max-w-lg space-y-4">
            <div className="space-y-1">
              <Input
                id="new-password"
                type="password"
                label={t('auth:profile.newPasswordLabel')}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                minLength={MIN_PASSWORD_LENGTH}
                autoComplete="new-password"
                aria-describedby="profile-password-hint"
                required
              />
              <p id="profile-password-hint" className="text-xs text-ds-text-secondary">
                {t('auth:setup.passwordHint')}
              </p>
            </div>

            <Input
              id="confirm-new-password"
              type="password"
              label={t('auth:profile.confirmNewPasswordLabel')}
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              minLength={MIN_PASSWORD_LENGTH}
              autoComplete="new-password"
              required
            />

            {error && (
              <p role="alert" className="text-xs text-error">
                {error}
              </p>
            )}

            <Button type="submit" disabled={isSubmitting} className="min-h-11 md:min-h-9">
              {isSubmitting ? t('auth:profile.updating') : t('auth:profile.updatePassword')}
            </Button>
          </form>
        </section>
      </div>
    </div>
  );
}
