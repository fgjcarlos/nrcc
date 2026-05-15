import { FormEvent, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { useAuth } from '../hooks/useAuth';
import { authService } from '../services/authService';

const MIN_PASSWORD_LENGTH = 8;

function formatDate(value?: string) {
  if (!value) return 'Not available';

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Not available';

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

export function ProfileView() {
  const { user } = useAuth();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const initials = useMemo(() => user?.username.slice(0, 2).toUpperCase() ?? 'NR', [user?.username]);

  if (!user) {
    return (
      <div className="p-6">
        <h1 className="text-2xl font-bold text-base-content">Profile</h1>
        <p className="mt-2 text-base-content/60">Sign in to view your account details.</p>
      </div>
    );
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);

    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`Password must be at least ${MIN_PASSWORD_LENGTH} characters long.`);
      return;
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }

    setIsSubmitting(true);
    try {
      await authService.changePassword(user.id, password);
      setPassword('');
      setConfirmPassword('');
      toast.success('Password updated successfully');
    } catch (err) {
      const message =
        (err as { response?: { data?: { error?: { message?: string } } } }).response?.data?.error?.message ||
        'Failed to update password';
      setError(message);
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/50">Account</p>
        <h1 className="mt-2 text-2xl font-bold text-base-content">Profile</h1>
        <p className="mt-2 text-sm text-base-content/60">
          Review your account details and update your own password.
        </p>
      </div>

      <section className="surface-card p-6">
        <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
          <div className="grid h-16 w-16 shrink-0 place-items-center rounded-2xl border border-border bg-primary/15 text-xl font-bold text-primary">
            {initials}
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-xl font-semibold text-base-content">{user.username}</h2>
            <p className="mt-1 text-sm capitalize text-base-content/60">{user.role}</p>
          </div>
        </div>

        <dl className="mt-6 grid gap-4 md:grid-cols-3">
          <div className="rounded-xl border border-border bg-base-200/45 p-4">
            <dt className="text-xs font-semibold uppercase tracking-wide text-base-content/50">User ID</dt>
            <dd className="mt-2 break-all text-sm text-base-content">{user.id}</dd>
          </div>
          <div className="rounded-xl border border-border bg-base-200/45 p-4">
            <dt className="text-xs font-semibold uppercase tracking-wide text-base-content/50">Created</dt>
            <dd className="mt-2 text-sm text-base-content">{formatDate(user.createdAt)}</dd>
          </div>
          <div className="rounded-xl border border-border bg-base-200/45 p-4">
            <dt className="text-xs font-semibold uppercase tracking-wide text-base-content/50">Last updated</dt>
            <dd className="mt-2 text-sm text-base-content">{formatDate(user.updatedAt)}</dd>
          </div>
        </dl>
      </section>

      <section className="surface-card p-6">
        <div className="mb-5">
          <h2 className="text-lg font-semibold text-base-content">Password</h2>
          <p className="mt-1 text-sm text-base-content/60">
            Choose a new password with at least {MIN_PASSWORD_LENGTH} characters.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="max-w-lg space-y-4">
          <div>
            <label htmlFor="new-password" className="mb-1 block text-sm font-medium text-base-content">
              New password
            </label>
            <input
              id="new-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
              minLength={MIN_PASSWORD_LENGTH}
              autoComplete="new-password"
              required
            />
          </div>

          <div>
            <label htmlFor="confirm-new-password" className="mb-1 block text-sm font-medium text-base-content">
              Confirm new password
            </label>
            <input
              id="confirm-new-password"
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
              minLength={MIN_PASSWORD_LENGTH}
              autoComplete="new-password"
              required
            />
          </div>

          {error && (
            <p role="alert" className="rounded-xl border border-error/30 bg-error/10 px-3 py-2 text-sm text-error">
              {error}
            </p>
          )}

          <button type="submit" className="action-btn-primary disabled:opacity-50" disabled={isSubmitting}>
            {isSubmitting ? 'Updating…' : 'Update password'}
          </button>
        </form>
      </section>
    </div>
  );
}
