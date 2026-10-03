import { useState } from 'react';
import { Plus, ShieldCheck, ShieldOff, Trash2, UserPlus } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { StatusChip } from '@/shared/components/ui';
import { cn } from '@/shared/lib';
import { useT } from '@/i18n';
import {
  applySecurityPatch,
  detectLegacyAliases,
  humanizeApplyError,
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

/** Per-user edits the operator has made since the last server fetch.
 *  Cleared automatically when the prop changes (server refetch). */
interface UserEdits {
  /** Username overrides indexed by user row index. */
  usernames: Map<number, string>;
  /** Permission overrides indexed by user row index. */
  permissions: Map<number, Permission>;
  /** Password overrides indexed by user row index (never re-populated from server). */
  passwords: Map<number, string>;
  /** Rows the operator added locally (not yet on the server). */
  added: AdminUser[];
  /** Server-user indices the operator has marked for deletion. */
  removed: Set<number>;
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
  const [edits, setEdits] = useState<UserEdits>({
    usernames: new Map(),
    permissions: new Map(),
    passwords: new Map(),
    added: [],
    removed: new Set(),
  });
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [appliedAt, setAppliedAt] = useState<number | null>(null);
  const [error, setError] = useState<string>();
  const legacyAliases = detectLegacyAliases(rawSettingsContent);

  // Server-authoritative user list (drives row count + base values).
  const serverUsers = config?.users ?? [];

  // Derived display values: server rows (minus any the operator marked
  // for deletion) with edit overrides applied + locally added rows
  // appended after.
  const displayUsers: AdminUser[] = [
    ...serverUsers
      .map((serverUser, index) => {
        if (edits.removed.has(index)) return null;
        const overrideUser: AdminUser = {
          username: edits.usernames.get(index) ?? serverUser.username ?? '',
          permissions:
            edits.permissions.get(index) ?? serverUser.permissions ?? '*',
          password: edits.passwords.get(index) ?? '',
        };
        return overrideUser;
      })
      .filter((u): u is AdminUser => u !== null),
    ...edits.added,
  ];
  const displayExpiry = edits.expiry ?? config?.sessionExpiryTime ?? 0;

  const surfacesConfigured = serverUsers.length > 0;
  const variant: 'success' | 'warning' | 'neutral' = !editable
    ? 'neutral'
    : surfacesConfigured
      ? 'success'
      : 'warning';

  const updateUsername = (index: number, value: string) => {
    setEdits((current) => {
      const next = new Map(current.usernames);
      next.set(index, value);
      return { ...current, usernames: next };
    });
    setError(undefined);
  };

  const updatePermission = (index: number, value: Permission) => {
    setEdits((current) => {
      const next = new Map(current.permissions);
      next.set(index, value);
      return { ...current, permissions: next };
    });
    setError(undefined);
  };

  const updatePassword = (index: number, value: string) => {
    setEdits((current) => {
      const next = new Map(current.passwords);
      next.set(index, value);
      return { ...current, passwords: next };
    });
    setError(undefined);
  };

  const updateExpiry = (value: number) => {
    setEdits((current) => ({ ...current, expiry: value }));
    setError(undefined);
  };

  const addUser = () => {
    setEdits((current) => ({
      ...current,
      added: [...current.added, { username: '', permissions: '*', password: '' }],
    }));
    setError(undefined);
  };

  const removeUser = (index: number) => {
    // Remove either an edited server user (mark the index as removed
    // + shift edits > index down by one so the remaining rows keep
    // their edits) or a locally-added row (drop the matching index
    // from edits.added). The two paths split cleanly on the server
    // row count.
    setEdits((current) => {
      const serverCount = serverUsers.length;
      if (index >= serverCount) {
        const addedIndex = index - serverCount;
        const newRemoved = new Set(current.removed);
        if (addedIndex < 0) {
          // Mark the corresponding server row as removed.
          newRemoved.add(index);
        }
        return {
          ...current,
          added: current.added.filter((_, i) => i !== addedIndex),
          removed: newRemoved,
        };
      }
      const shiftMap = <Value,>(src: Map<number, Value>) => {
        const out = new Map<number, Value>();
        for (const [k, v] of src.entries()) {
          if (k < index) out.set(k, v);
          else if (k > index) out.set(k - 1, v);
        }
        return out;
      };
      const newRemoved = new Set<number>();
      for (const r of current.removed) {
        if (r === index) continue;
        newRemoved.add(r < index ? r : r - 1);
      }
      newRemoved.add(index);
      return {
        ...current,
        usernames: shiftMap(current.usernames),
        permissions: shiftMap(current.permissions),
        passwords: shiftMap(current.passwords),
        removed: newRemoved,
      };
    });
    setError(undefined);
  };

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
      setEdits({
        usernames: new Map(),
        permissions: new Map(),
        passwords: new Map(),
        added: [],
        removed: new Set(),
      });
      // The lint rule flags Date.now() as impure; this call runs from
      // the Save button click handler (an event), not from render, so
      // purity is preserved. Capture once to keep the call out of
      // the setState callback.
      // eslint-disable-next-line react-hooks/purity
      const ts = Date.now();
      setAppliedAt(ts);
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
            value={displayExpiry}
            onChange={(event) => updateExpiry(Number(event.target.value))}
          />
        </label>

        {displayUsers.length === 0 && (
          <p
            className="rounded-xl border border-border bg-base-200/40 px-4 py-3 text-sm text-base-content/65"
            data-testid="boundary-admin-auth-empty"
          >
            {t('security:adminAuthBoundary.noUsers')}
          </p>
        )}

        {displayUsers.map((user, index) => (
          <div
            key={index}
            className="grid gap-3 rounded-xl border border-border p-3 md:grid-cols-4"
            data-testid="boundary-admin-auth-user-row"
          >
            <label className="text-sm">
              {t('security:adminAuthBoundary.username')}
              <input
                className="input input-bordered mt-1 w-full"
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
                className="select select-bordered mt-1 w-full"
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
                className="input input-bordered mt-1 w-full"
                type="password"
                value={user.password}
                data-testid="boundary-admin-auth-password"
                onChange={(event) => updatePassword(index, event.target.value)}
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
