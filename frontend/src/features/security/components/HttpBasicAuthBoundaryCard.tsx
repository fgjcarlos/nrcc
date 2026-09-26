import { useEffect, useState } from 'react';
import { Check, Lock, Unlock } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import {
  applySecurityPatch,
  detectLegacyAliases,
  humanizeApplyError,
  isBcryptOrEmpty,
  reportApplyError,
  reportApplySuccess,
} from './securityCenterHelpers';

type BasicAuthForm = { user: string; pass: string; storedPass?: string };

/**
 * Loose config shape — accept whatever the backend sends. The
 * NodeRedConfigResponse type tags adminAuth / nodeHttpAuth / staticAuth
 * as CredentialsAuthResponse, which doesn't include the `user` and
 * `pass` keys the basic-auth surface actually carries at runtime.
 * Accepting only `user` + `pass` here keeps the public prop honest
 * while letting SecurityView forward the runtime payload without
 * unsafe casts at the boundary.
 */
export type HttpBasicAuthSurface = { user?: string; pass?: string } | null;

interface HttpBasicAuthBoundaryCardProps {
  /** Which boundary this card represents. Used to namespace the test ids,
   *  aria-labels, save action label, and the i18n keys. */
  surface: 'httpNodeAuth' | 'httpStaticAuth';
  config: HttpBasicAuthSurface;
  rawSettingsContent: string;
  expectedRevision?: string;
  editable: boolean;
  onApplied: () => void;
}

/**
 * HttpBasicAuthBoundaryCard — issue #766 slice E
 *
 * Second (or third) of the four security-surface boundary cards.
 * Renders the httpNodeAuth or httpStaticAuth basic-auth form.
 *
 * Each surface is a thin wrapper over the same body so we can keep
 * the migration / bcrypt-hash / apply pipeline in one place.
 */
export function HttpBasicAuthBoundaryCard({
  surface,
  config,
  rawSettingsContent,
  expectedRevision,
  editable,
  onApplied,
}: HttpBasicAuthBoundaryCardProps) {
  const { t } = useT();
  const [value, setValue] = useState<BasicAuthForm>({ user: '', pass: '' });
  const [saving, setSaving] = useState(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [error, setError] = useState<string>();
  const [appliedAt, setAppliedAt] = useState<number | null>(null);
  const [dirty, setDirty] = useState(false);

  const surfaceConfigured = Boolean(config?.user);
  const legacyAliases = detectLegacyAliases(rawSettingsContent);
  // A legacy alias for httpNodeAuth/staticAuth lives in rawSettingsContent but
  // the surface itself may have no entry yet — that's the migration case.
  const isLegacySurface =
    (surface === 'httpNodeAuth' &&
      legacyAliases.includes('nodeHttpAuth') &&
      !config?.user) ||
    (surface === 'httpStaticAuth' &&
      legacyAliases.includes('staticAuth') &&
      !config?.user);
  const variant: 'success' | 'warning' | 'neutral' = !editable
    ? 'neutral'
    : surfaceConfigured
      ? 'success'
      : isLegacySurface
        ? 'warning'
        : 'neutral';

  useEffect(() => {
    setValue({
      user: config?.user ?? '',
      pass: '',
      storedPass: config?.pass,
    });
    setDirty(false);
    setError(undefined);
  }, [config]);

  const i18n = (suffix: string) => `security:${surface}.${suffix}`;

  const markDirty = () => {
    setDirty(true);
    setError(undefined);
  };

  const apply = async () => {
    setConfirmMigration(false);
    setSaving(true);
    setError(undefined);

    const patch =
      value.user.trim().length === 0
        ? { [surface]: null }
        : {
            [surface]: {
              user: value.user.trim(),
              pass: value.pass || value.storedPass || '',
            },
          };

    const result = await applySecurityPatch(patch, expectedRevision);
    setSaving(false);
    if (result.status === 'applied') {
      setDirty(false);
      setAppliedAt(Date.now());
      reportApplySuccess(t(i18n('applied')));
      onApplied();
    } else {
      setError(humanizeApplyError(result.error?.status));
      reportApplyError({ status: 'FAILED', detail: result.error?.detail });
    }
  };

  const save = async () => {
    if (isLegacySurface) {
      setConfirmMigration(true);
      return;
    }
    await apply();
  };

  const typedPass = value.pass.trim();
  const bcryptInvalid = typedPass.length > 0 && !isBcryptOrEmpty(typedPass);

  return (
    <article
      data-testid={`boundary-${surface}`}
      aria-labelledby={`boundary-${surface}-title`}
      className="card surface-card border border-border p-6"
    >
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {surfaceConfigured ? (
            <Lock className="h-6 w-6 text-success" aria-hidden="true" />
          ) : (
            <Unlock className="h-6 w-6 text-warning" aria-hidden="true" />
          )}
          <div className="min-w-0">
            <h2
              id={`boundary-${surface}-title`}
              className="text-base font-semibold text-base-content"
            >
              {t(i18n('title'))}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {t(i18n('subtitle'))}
            </p>
          </div>
        </div>
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t(i18n('statusAria'), {
            state: surfaceConfigured ? 'on' : 'off',
          })}
        >
          {surfaceConfigured
            ? t(i18n('statusOn'))
            : t(i18n('statusOff'))}
        </StatusChip>
      </header>

      {!editable && (
        <p
          role="status"
          className="mt-3 rounded-xl border border-warning/40 bg-warning/10 p-3 text-sm text-warning"
          data-testid={`boundary-${surface}-readonly`}
        >
          {t(i18n('readOnly'))}
        </p>
      )}

      {error && (
        <p
          role="status"
          aria-live="polite"
          className="mt-3 rounded-xl border border-danger/40 bg-danger/10 p-3 text-sm text-danger"
          data-testid={`boundary-${surface}-error`}
        >
          {error}
        </p>
      )}

      {bcryptInvalid && (
        <p
          role="status"
          className="mt-3 rounded-xl border border-warning/40 bg-warning/10 p-3 text-sm text-warning"
          data-testid={`boundary-${surface}-bcrypt-warning`}
        >
          {t(i18n('bcryptWarning'))}
        </p>
      )}

      {isLegacySurface && editable && (
        <div
          role="status"
          className="mt-3 rounded-xl border border-warning/40 bg-warning/10 p-3 text-sm"
          data-testid={`boundary-${surface}-legacy`}
        >
          <h3 className="font-medium">{t(i18n('legacyTitle'))}</h3>
          <p className="text-base-content/85">
            {t(i18n('legacyBody'), {
              alias: surface === 'httpNodeAuth' ? 'nodeHttpAuth' : 'staticAuth',
            })}
          </p>
        </div>
      )}

      <fieldset
        disabled={!editable || saving}
        className="mt-5 space-y-4"
        data-testid={`boundary-${surface}-fieldset`}
      >
        <label className="text-sm">
          {t(i18n('username'))}
          <input
            className="input input-bordered mt-1 w-full"
            data-testid={`boundary-${surface}-user`}
            value={value.user}
            onChange={(event) => {
              markDirty();
              setValue((current) => ({ ...current, user: event.target.value }));
            }}
          />
        </label>
        <label className="text-sm">
          {t(i18n('bcryptHash'))}
          <input
            className="input input-bordered mt-1 w-full"
            data-testid={`boundary-${surface}-pass`}
            type="password"
            value={value.pass}
            onChange={(event) => {
              markDirty();
              setValue((current) => ({ ...current, pass: event.target.value }));
            }}
          />
          {value.storedPass && (
            <p
              className="mt-1 text-xs text-base-content/55"
              data-testid={`boundary-${surface}-stored-hash`}
            >
              {t(i18n('storedHashHint'), { hash: value.storedPass })}
            </p>
          )}
          <p className="mt-1 text-sm text-base-content/65">
            {t(i18n('bcryptHint'))}
          </p>
        </label>

        <button
          type="button"
          className="action-btn-primary flex items-center gap-2"
          onClick={save}
          disabled={
            !editable || saving || (!dirty && !isLegacySurface)
          }
          data-testid={`boundary-${surface}-save`}
        >
          <Check className="h-4 w-4" aria-hidden="true" />
          {saving ? t(i18n('saving')) : t(i18n('save'))}
        </button>
      </fieldset>

      {appliedAt && !error && (
        <p
          role="status"
          aria-live="polite"
          className="mt-2 text-sm text-success"
          data-testid={`boundary-${surface}-applied`}
        >
          {t(i18n('appliedLabel'))}
        </p>
      )}

      <ConfirmationDialog
        isOpen={confirmMigration}
        title={t(i18n('migrationTitle'))}
        description={t(i18n('migrationDescription'), {
          alias: surface === 'httpNodeAuth' ? 'nodeHttpAuth' : 'staticAuth',
        })}
        acknowledgement={t(i18n('migrationAck'))}
        variant="warning"
        onCancel={() => setConfirmMigration(false)}
        onConfirm={apply}
      />
    </article>
  );
}
