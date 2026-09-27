import { useEffect, useState } from 'react';
import { Plus, ShieldCheck, ShieldOff, Trash2, UserPlus } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { StatusChip } from '@/shared/components/ui';
import { cn } from '@/shared/lib';
import { useT } from '@/i18n';
import {
  applySecurityPatch,
  detectLegacyAliases,
  humanizeApplyError,
  isBcryptOrEmpty,
  reportApplyError,
  reportApplySuccess,
} from './securityCenterHelpers';

type AdminUser = { username: string; permissions: '*' | 'read'; password: string };
type Permission = AdminUser['permissions'];

interface AdminAuthConfig {
  type?: 'credentials';
  users?: Array<{ username?: string; permissions?: Permission }>;
  sessionExpiryTime?: number;
}

interface AdminAuthBoundaryCardProps {
  config: AdminAuthConfig | null;
  rawSettingsContent: string;
  expectedRevision?: string;
  editable: boolean;
  onApplied: () => void;
}

const emptyUser = (): AdminUser => ({ username: '', permissions: '*', password: '' });

/**
 * AdminAuthBoundaryCard — issue #766 slice E
 *
 * First of the four security-surface boundary cards. Owns the
 * `adminAuth` users + `sessionExpiryTime` patch via POST
 * /api/config/apply. The form is fully read-only when `editable` is
 * false (Node-RED 4 / unknown runtime).
 *
 * Save button stays mounted regardless of the dirty/clean state so the
 * operator always sees the action; only the enabled state changes.
 */
export function AdminAuthBoundaryCard({
  config,
  rawSettingsContent,
  expectedRevision,
  editable,
  onApplied,
}: AdminAuthBoundaryCardProps) {
  const { t } = useT();
  // Derive users / expiry directly from the prop. Combined with the
  // useEffect below (which only runs once on mount), this avoids the
  // race where the first render uses useState's lazy initialiser
  // (config prop still null) and the second render keeps the stale
  // state. By keeping the working copy of users + expiry in local
  // state — and reseeding it on every config change — every render
  // shows the latest server values.
  const [users, setUsers] = useState<AdminUser[]>(() =>
    config?.users?.map((user) => ({
      username: user.username ?? '',
      permissions: user.permissions ?? '*',
      password: '',
    })) ?? [],
  );
  const [expiry, setExpiry] = useState<number>(() => config?.sessionExpiryTime ?? 0);
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [appliedAt, setAppliedAt] = useState<number | null>(null);
  const [error, setError] = useState<string>();
  const legacyAliases = detectLegacyAliases(rawSettingsContent);
  const surfacesConfigured = (config?.users?.length ?? 0) > 0;
  const variant: 'success' | 'warning' | 'neutral' = !editable
    ? 'neutral'
    : surfacesConfigured
      ? 'success'
      : 'warning';

  useEffect(() => {
    setUsers(
      config?.users?.map((user) => ({
        username: user.username ?? '',
        permissions: user.permissions ?? '*',
        password: '',
      })) ?? [],
    );
    setExpiry(config?.sessionExpiryTime ?? 0);
    setError(undefined);
  }, [config]);

  const updateUser = (index: number, update: Partial<AdminUser>) => {
    setUsers((current) =>
      current.map((user, i) => (i === index ? { ...user, ...update } : user)),
    );
  };

  const apply = async () => {
    setConfirmMigration(false);
    setSaving(true);
    setError(undefined);

    const adminAuth =
      users.length === 0
        ? null
        : {
            type: 'credentials' as const,
            users: users.map(({ username, permissions, password }) => ({
              username,
              permissions,
              password,
            })),
            ...(expiry > 0 ? { sessionExpiryTime: expiry } : {}),
          };

    const result = await applySecurityPatch({ adminAuth }, expectedRevision);
    setSaving(false);
    if (result.status === 'applied') {
      setAppliedAt(Date.now());
      reportApplySuccess(t('security:adminAuthBoundary.applied'));
      onApplied();
    } else {
      setError(humanizeApplyError(result.error?.status));
      reportApplyError({ status: 'FAILED', detail: result.error?.detail });
    }
  };

  const save = async () => {
    if (legacyAliases.length > 0) {
      setConfirmMigration(true);
      return;
    }
    await apply();
  };

  const addUser = () => {
    setUsers((current) => [...current, emptyUser()]);
  };

  const removeUser = (index: number) => {
    setUsers((current) => current.filter((_, i) => i !== index));
  };

  const disabled = !editable || saving;

  return (
    <article
      data-testid="boundary-admin-auth"
      aria-labelledby="boundary-admin-auth-title"
      className="card surface-card border border-border p-6"
    >
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {surfacesConfigured ? (
            <ShieldCheck className="h-6 w-6 text-success" aria-hidden="true" />
          ) : (
            <ShieldOff className="h-6 w-6 text-warning" aria-hidden="true" />
          )}
          <div className="min-w-0">
            <h2
              id="boundary-admin-auth-title"
              className="text-base font-semibold text-base-content"
            >
              {t('security:adminAuthBoundary.title')}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {t('security:adminAuthBoundary.subtitle')}
            </p>
          </div>
        </div>
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t('security:adminAuthBoundary.statusAria', {
            state: surfacesConfigured ? 'on' : 'off',
          })}
        >
          {surfacesConfigured
            ? t('security:adminAuthBoundary.statusOn')
            : t('security:adminAuthBoundary.statusOff')}
        </StatusChip>
      </header>

      {!editable && (
        <p
          role="status"
          className="mt-3 rounded-xl border border-warning/40 bg-warning/10 p-3 text-sm text-warning"
          data-testid="boundary-admin-auth-readonly"
        >
          {t('security:adminAuthBoundary.readOnly')}
        </p>
      )}

      {error && (
        <p
          role="status"
          aria-live="polite"
          className="mt-3 rounded-xl border border-danger/40 bg-danger/10 p-3 text-sm text-danger"
          data-testid="boundary-admin-auth-error"
        >
          {error}
        </p>
      )}

      {legacyAliases.length > 0 && editable && (
        <div
          role="status"
          className="mt-3 rounded-xl border border-warning/40 bg-warning/10 p-3 text-sm"
          data-testid="boundary-admin-auth-legacy"
        >
          <h3 className="font-medium">{t('security:adminAuthBoundary.legacyTitle')}</h3>
          <p className="text-base-content/85">
            {t('security:adminAuthBoundary.legacyBody', {
              aliases: legacyAliases
                .map((name) => (name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth'))
                .join(' and '),
            })}
          </p>
        </div>
      )}

      <fieldset
        disabled={disabled}
        className="mt-5 space-y-4"
        data-testid="boundary-admin-auth-fieldset"
      >
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-sm font-medium">{t('security:adminAuthBoundary.usersLabel')}</h3>
          <button
            type="button"
            className="action-btn-secondary"
            onClick={addUser}
            data-testid="boundary-admin-auth-add"
          >
            <UserPlus className="h-4 w-4" aria-hidden="true" />
            {t('security:adminAuthBoundary.addUser')}
          </button>
        </div>

        <label className="block text-sm">
          {t('security:adminAuthBoundary.sessionExpiry')}
          <input
            aria-label={t('security:adminAuthBoundary.sessionExpiryLabel')}
            data-testid="boundary-admin-auth-expiry"
            className="input input-bordered mt-1 w-full"
            min={0}
            type="number"
            value={expiry}
            onChange={(event) => {
              setExpiry(Number(event.target.value));
            }}
          />
        </label>

        {users.length === 0 && (
          <p
            className="rounded-xl border border-border bg-base-200/40 px-4 py-3 text-sm text-base-content/65"
            data-testid="boundary-admin-auth-empty"
          >
            {t('security:adminAuthBoundary.noUsers')}
          </p>
        )}

        {users.map((user, index) => (
          <div
            key={`${user.username}-${index}`}
            className="grid gap-3 rounded-xl border border-border p-3 md:grid-cols-4"
            data-testid="boundary-admin-auth-user-row"
          >
            <label className="text-sm">
              {t('security:adminAuthBoundary.username')}
              <input
                className="input input-bordered mt-1 w-full"
                data-testid="boundary-admin-auth-username"
                value={user.username}
                onChange={(event) => updateUser(index, { username: event.target.value })}
              />
            </label>
            <label className="text-sm">
              {t('security:adminAuthBoundary.permission')}
              <select
                aria-label={
                  user.username
                    ? t('security:adminAuthBoundary.permissionFor', { username: user.username })
                    : t('security:adminAuthBoundary.permissionForIndex', { index: index + 1 })
                }
                className="select select-bordered mt-1 w-full"
                value={user.permissions}
                onChange={(event) =>
                  updateUser(index, { permissions: event.target.value as Permission })
                }
                data-testid="boundary-admin-auth-permission"
              >
                <option value="*">{t('security:adminAuthBoundary.fullAccess')}</option>
                <option value="read">
                  {t('security:adminAuthBoundary.readOnlyPermission')}
                </option>
              </select>
            </label>
            <label className="text-sm">
              {t('security:adminAuthBoundary.newPassword')}
              <input
                className="input input-bordered mt-1 w-full"
                type="password"
                value={user.password}
                data-testid="boundary-admin-auth-password"
                onChange={(event) => updateUser(index, { password: event.target.value })}
              />
            </label>
            <button
              type="button"
              className={cn(
                'btn btn-ghost self-end',
                'border border-border/60 bg-base-200/40 text-base-content/80',
              )}
              aria-label={t('security:adminAuthBoundary.removeUser', {
                username: user.username || `user ${index + 1}`,
              })}
              data-testid="boundary-admin-auth-remove"
              onClick={() => removeUser(index)}
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
              {t('security:adminAuthBoundary.remove')}
            </button>
          </div>
        ))}

        <button
          type="button"
          className="action-btn-primary flex items-center gap-2"
          onClick={save}
          disabled={disabled}
          data-testid="boundary-admin-auth-save"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          {saving
            ? t('security:adminAuthBoundary.saving')
            : t('security:adminAuthBoundary.save')}
        </button>
      {appliedAt && !error && (
        <p
          role="status"
          aria-live="polite"
          className="mt-2 text-sm text-success"
          data-testid="boundary-admin-auth-applied"
        >
          {t('security:adminAuthBoundary.appliedLabel')}
        </p>
      )}
      </fieldset>

      <ConfirmationDialog
        isOpen={confirmMigration}
        title={t('security:adminAuthBoundary.migrationTitle')}
        description={t('security:adminAuthBoundary.migrationDescription', {
          aliases: legacyAliases
            .map((name) => (name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth'))
            .join(', '),
        })}
        acknowledgement={t('security:adminAuthBoundary.migrationAck')}
        variant="warning"
        onCancel={() => setConfirmMigration(false)}
        onConfirm={apply}
      />
    </article>
  );
}

/** Helper kept exported so the test can exercise the bcrypt check. */
export { isBcryptOrEmpty as _isBcryptOrEmpty };
