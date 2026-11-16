import { useEffect, useState } from 'react';
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

const ANCHORS = { httpNodeAuth: 'http-node-auth', httpStaticAuth: 'http-static-auth' } as const;

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
 * the migration / password / apply pipeline in one place. A plain
 * password is sent as typed; the backend stores it as a bcrypt hash.
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
  // Off is neutral (often off on purpose); a pending legacy alias is amber.
  const state = surfaceConfigured ? 'statusOn' : isLegacySurface ? 'statusLegacy' : 'statusOff';
  const variant = surfaceConfigured ? 'success' : isLegacySurface ? 'warning' : 'neutral';

  useEffect(() => {
    setValue({
      user: config?.user ?? '',
      pass: '',
      storedPass: config?.pass,
    });
    setError(undefined);
  }, [config]);

  const i18n = (suffix: string) => `security:${surface}.${suffix}`;

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
      // The lint rule flags Date.now() as impure; this call runs from
      // the Save button click handler (an event), not from render, so
      // purity is preserved. Capture once to keep the call out of
      // the setState callback.
      const ts = Date.now();
      setAppliedAt(ts);
      reportApplySuccess(t(i18n('applied')));
      onApplied();
    } else {
      const message = t(applyErrorKey(result.error?.status));
      setError(message);
      reportApplyError(message);
    }
  };

  const save = async () => {
    if (isLegacySurface) {
      setConfirmMigration(true);
      return;
    }
    await apply();
  };

  return (
    <>
    <SecuritySurface
      anchor={ANCHORS[surface]}
      testId={`boundary-${surface}`}
      title={t(i18n('title'))}
      summary={t(i18n('subtitle'))}
      status={
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t(i18n('statusAria'), { state: t(i18n(state)) })}
        >
          {t(i18n(state))}
        </StatusChip>
      }
    >
      {!editable && (
        <p
          role="status"
          className="mb-4 rounded-box border border-ds-warning/40 bg-ds-warning/10 px-3 py-2 text-sm text-ds-on-tint-warning"
          data-testid={`boundary-${surface}-readonly`}
        >
          {t(i18n('readOnly'))}
        </p>
      )}

      {error && (
        <p
          role="status"
          aria-live="polite"
          className="mb-4 rounded-box border border-ds-danger/40 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error"
          data-testid={`boundary-${surface}-error`}
        >
          {error}
        </p>
      )}

      {isLegacySurface && editable && (
        <div
          role="status"
          className="mb-4 rounded-box border border-ds-warning/40 bg-ds-warning/10 px-3 py-2 text-sm"
          data-testid={`boundary-${surface}-legacy`}
        >
          <h3 className="font-semibold">{t(i18n('legacyTitle'))}</h3>
          <p className="text-ds-text-secondary">
            {t(i18n('legacyBody'), {
              alias: surface === 'httpNodeAuth' ? 'nodeHttpAuth' : 'staticAuth',
            })}
          </p>
        </div>
      )}

      <fieldset
        disabled={!editable || saving}
        className="grid max-w-2xl gap-4 sm:grid-cols-2"
        data-testid={`boundary-${surface}-fieldset`}
      >
        <label className="text-sm">
          {t(i18n('username'))}
          <input
            className="input input-bordered mt-1 min-h-11 w-full sm:min-h-0"
            data-testid={`boundary-${surface}-user`}
            value={value.user}
            onChange={(event) => {
              setValue((current) => ({ ...current, user: event.target.value }));
            }}
          />
        </label>
        <label className="text-sm">
          {t(i18n('password'))}
          <input
            className="input input-bordered mt-1 min-h-11 w-full sm:min-h-0"
            data-testid={`boundary-${surface}-pass`}
            type="password"
            value={value.pass}
            onChange={(event) => {
              setValue((current) => ({ ...current, pass: event.target.value }));
            }}
          />
          {value.storedPass && (
            <p
              className="mt-1 break-all font-mono text-xs text-ds-text-muted"
              data-testid={`boundary-${surface}-stored-hash`}
            >
              {t(i18n('storedHashHint'), { hash: value.storedPass })}
            </p>
          )}
        </label>

        <div className="flex flex-wrap items-center gap-3 sm:col-span-2">
          <button
            type="button"
            className="action-btn-primary min-h-11 sm:min-h-0"
            onClick={save}
            disabled={!editable || saving}
            data-testid={`boundary-${surface}-save`}
          >
            {saving ? t(i18n('saving')) : t(i18n('save'))}
          </button>
          {appliedAt && !error && (
            <p
              role="status"
              aria-live="polite"
              className="text-sm text-ds-on-tint-success"
              data-testid={`boundary-${surface}-applied`}
            >
              {t(i18n('appliedLabel'))}
            </p>
          )}
        </div>
      </fieldset>
    </SecuritySurface>

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
    </>
  );
}
