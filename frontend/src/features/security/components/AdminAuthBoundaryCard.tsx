import { useState } from 'react';
import { Trash2, UserPlus } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import {
  applySecurityPatch,
  applyErrorKey,
  detectLegacyAliases,
  reportApplyError,
  reportApplySuccess,
} from './securityCenterHelpers';
import { SecuritySurface } from './SecuritySurface';

type AdminUser = { username: string; permissions: '*' | 'read'; password: string };
type Permission = AdminUser['permissions'];

interface AdminAuthConfig {
  type?: string;
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

/** Edits the operator has made since the last successful apply. */
interface UserEdits {
  /** Full edited user list (server rows + added rows, minus removed);
   *  undefined until the first user edit so server values drive the form. */
  users?: AdminUser[];
  /** Expiry override (when operator typed a new value). */
  expiry?: number;
}

/**
 * AdminAuthBoundaryCard — issue #766 slice E
 *
 * First of the four security-surface boundary cards. Owns the
 * `adminAuth` users + `sessionExpiryTime` patch via POST /api/config/apply.
 *
 * State model: the authoritative server values come from the
 * `config` prop on every render. Local state only tracks the
 * operator's in-flight edits (typed usernames, typed passwords, new
 * expiry) and is reset whenever the prop changes. This avoids the
 * race where a useEffect lag between first render and the config
 * query resolving would leave the form showing empty users /
 * default expiry even though config.users.length is > 0.
 */
export function AdminAuthBoundaryCard({
  config,
  rawSettingsContent,
  expectedRevision,
  editable,
  onApplied,
}: AdminAuthBoundaryCardProps) {
  const { t } = useT();
  const [edits, setEdits] = useState<UserEdits>({});
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [appliedAt, setAppliedAt] = useState<number | null>(null);
  const [error, setError] = useState<string>();
  const legacyAliases = detectLegacyAliases(rawSettingsContent);

  // Server-authoritative user list (drives row count + base values).
  const serverUsers = config?.users ?? [];

  // Server rows start with an empty password (empty keeps the stored hash).
  const displayUsers: AdminUser[] =
    edits.users ??
    serverUsers.map((serverUser) => ({
      username: serverUser.username ?? '',
      permissions: serverUser.permissions ?? '*',
      password: '',
    }));
  const displayExpiry = edits.expiry ?? config?.sessionExpiryTime ?? 0;

  const surfacesConfigured = serverUsers.length > 0;
  // Off is amber here: without adminAuth anyone reaching the editor can deploy.
  const variant = surfacesConfigured ? 'success' : 'warning';

  const updateUsers = (change: (users: AdminUser[]) => AdminUser[]) => {
    setEdits((current) => ({ ...current, users: change(current.users ?? displayUsers) }));
    setError(undefined);
  };

  const updateUser = (index: number, patch: Partial<AdminUser>) =>
    updateUsers((users) =>
      users.map((user, i) => (i === index ? { ...user, ...patch } : user)),
    );

  const updateUsername = (index: number, value: string) =>
    updateUser(index, { username: value });

  const updatePermission = (index: number, value: Permission) =>
    updateUser(index, { permissions: value });

  const updatePassword = (index: number, value: string) =>
    updateUser(index, { password: value });

  const updateExpiry = (value: number) => {
    setEdits((current) => ({ ...current, expiry: value }));
    setError(undefined);
  };

  const addUser = () =>
    updateUsers((users) => [...users, { username: '', permissions: '*', password: '' }]);

  const removeUser = (index: number) =>
    updateUsers((users) => users.filter((_, i) => i !== index));

  const apply = async () => {
    setConfirmMigration(false);
    setSaving(true);
    setError(undefined);

    const adminAuth =
      displayUsers.length === 0
        ? null
        : {
            type: 'credentials' as const,
            users: displayUsers.map(({ username, permissions, password }) => ({
              username,
              permissions,
              password,
            })),
            ...(displayExpiry > 0 ? { sessionExpiryTime: displayExpiry } : {}),
          };

    const result = await applySecurityPatch({ adminAuth }, expectedRevision);
    setSaving(false);
    if (result.status === 'applied') {
      setEdits({});
      setAppliedAt(Date.now());
      reportApplySuccess(t('security:adminAuthBoundary.applied'));
      onApplied();
    } else {
      const message = t(applyErrorKey(result.error?.status));
      setError(message);
      reportApplyError(message);
    }
  };

  const save = async () => {
    if (legacyAliases.length > 0) {
      setConfirmMigration(true);
      return;
    }
    await apply();
  };

  const disabled = !editable || saving;

  const canonicalAliases = legacyAliases
    .map((name) => (name === 'nodeHttpAuth' ? 'httpNodeAuth' : 'httpStaticAuth'))
    .join(', ');
  const title = t('security:adminAuthBoundary.title');

  return (
    <>
    <SecuritySurface
      anchor="admin-auth"
      testId="boundary-admin-auth"
      title={title}
      summary={t('security:adminAuthBoundary.subtitle')}
      status={
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t('security:adminAuthBoundary.statusAria', {
            state: surfacesConfigured
              ? t('security:adminAuthBoundary.statusOn')
              : t('security:adminAuthBoundary.statusOff'),
          })}
        >
          {surfacesConfigured
            ? t('security:adminAuthBoundary.statusOn')
            : t('security:adminAuthBoundary.statusOff')}
        </StatusChip>
      }
    >
      {!editable && (
        <p
          role="status"
          className="mb-4 rounded-box border border-ds-warning/40 bg-ds-warning/10 px-3 py-2 text-sm text-ds-on-tint-warning"
          data-testid="boundary-admin-auth-readonly"
        >
          {t('security:adminAuthBoundary.readOnly')}
        </p>
      )}

      {error && (
        <p
          role="status"
          aria-live="polite"
          className="mb-4 rounded-box border border-ds-danger/40 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error"
          data-testid="boundary-admin-auth-error"
        >
          {error}
        </p>
      )}

      {legacyAliases.length > 0 && editable && (
        <div
          role="status"
          className="mb-4 rounded-box border border-ds-warning/40 bg-ds-warning/10 px-3 py-2 text-sm"
          data-testid="boundary-admin-auth-legacy"
        >
          <h3 className="font-semibold">{t('security:adminAuthBoundary.legacyTitle')}</h3>
          <p className="text-ds-text-secondary">
            {t('security:adminAuthBoundary.legacyBody', { aliases: canonicalAliases })}
          </p>
        </div>
      )}

      <fieldset
        disabled={disabled}
        className="space-y-4"
        data-testid="boundary-admin-auth-fieldset"
      >
        <label className="block max-w-xs text-sm">
          {t('security:adminAuthBoundary.sessionExpiry')}
          <input
            aria-label={t('security:adminAuthBoundary.sessionExpiryLabel')}
            data-testid="boundary-admin-auth-expiry"
            className="input input-bordered mt-1 min-h-11 w-full md:min-h-0"
            min={0}
            type="number"
            value={displayExpiry}
            onChange={(event) => updateExpiry(Number(event.target.value))}
          />
        </label>

        <div className="flex items-center justify-between gap-3 border-b border-ds-border-subtle pb-2">
          <h3 className="text-sm font-semibold">{t('security:adminAuthBoundary.usersLabel')}</h3>
          <button
            type="button"
            className="action-btn-secondary min-h-11 md:min-h-0"
            onClick={addUser}
            data-testid="boundary-admin-auth-add"
          >
            <UserPlus className="h-4 w-4" aria-hidden="true" />
            {t('security:adminAuthBoundary.addUser')}
          </button>
        </div>

        {displayUsers.length === 0 && (
          <p
            className="text-sm text-ds-text-secondary"
            data-testid="boundary-admin-auth-empty"
          >
            {t('security:adminAuthBoundary.noUsers')}
          </p>
        )}

        {displayUsers.map((user, index) => (
          <div
            key={index}
            className="grid gap-3 border-b border-ds-border-subtle pb-4 md:grid-cols-[1fr_1fr_1fr_auto]"
            data-testid="boundary-admin-auth-user-row"
          >
            <label className="text-sm">
              {t('security:adminAuthBoundary.username')}
              <input
                className="input input-bordered mt-1 min-h-11 w-full md:min-h-0"
                data-testid="boundary-admin-auth-username"
                value={user.username}
                onChange={(event) => updateUsername(index, event.target.value)}
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
                className="select select-bordered mt-1 min-h-11 w-full md:min-h-0"
                value={user.permissions}
                onChange={(event) =>
                  updatePermission(index, event.target.value as Permission)
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
                className="input input-bordered mt-1 min-h-11 w-full md:min-h-0"
                type="password"
                value={user.password}
                data-testid="boundary-admin-auth-password"
                onChange={(event) => updatePassword(index, event.target.value)}
              />
            </label>
            <button
              type="button"
              className="action-btn-ghost min-h-11 self-end md:min-h-0"
              aria-label={t('security:adminAuthBoundary.removeUser', {
                username:
                  user.username ||
                  t('security:adminAuthBoundary.userFallback', { index: index + 1 }),
              })}
              data-testid="boundary-admin-auth-remove"
              onClick={() => removeUser(index)}
            >
              <Trash2 className="h-4 w-4" aria-hidden="true" />
              {t('security:adminAuthBoundary.remove')}
            </button>
          </div>
        ))}

        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            className="action-btn-primary min-h-11 md:min-h-0"
            onClick={save}
            disabled={disabled}
            data-testid="boundary-admin-auth-save"
          >
            {saving
              ? t('security:adminAuthBoundary.saving')
              : t('security:adminAuthBoundary.save')}
          </button>
          {appliedAt && !error && (
            <p
              role="status"
              aria-live="polite"
              className="text-sm text-ds-on-tint-success"
              data-testid="boundary-admin-auth-applied"
            >
              {t('security:adminAuthBoundary.appliedLabel')}
            </p>
          )}
        </div>
      </fieldset>
    </SecuritySurface>

      <ConfirmationDialog
        isOpen={confirmMigration}
        title={t('security:adminAuthBoundary.migrationTitle')}
        description={t('security:adminAuthBoundary.migrationDescription', {
          aliases: canonicalAliases,
        })}
        acknowledgement={t('security:adminAuthBoundary.migrationAck')}
        variant="warning"
        onCancel={() => setConfirmMigration(false)}
        onConfirm={apply}
      />
    </>
  );
}
