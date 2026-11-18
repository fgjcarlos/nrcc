import { useEffect, useRef, useState } from 'react';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import { ReviewChangesPanel } from '@/features/configuration/components/ReviewChangesPanel';
import {
  applySecurityPatch,
  applyErrorKey,
  detectLegacyAliases,
} from './securityCenterHelpers';
import { SecuritySurface } from './SecuritySurface';
import { basicAuthReview, migrationRows } from '../lib/securityReview';
import { useSurfaceApply } from '../lib/useSurfaceApply';

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
  // Synchronous guard: a same-tick double click must not start two applies.
  const savingRef = useRef(false);
  const [confirmMigration, setConfirmMigration] = useState(false);
  const [error, setError] = useState<string>();

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
  const legacyAlias = surface === 'httpNodeAuth' ? 'nodeHttpAuth' : 'staticAuth';
  const reviewEntries = [
    ...migrationRows(isLegacySurface ? [legacyAlias] : [], t),
    ...basicAuthReview(surface, config, value, t),
  ];
  const review = useSurfaceApply(reviewEntries.length, value, editable);

  const edit = (patch: Partial<BasicAuthForm>) => {
    setValue((current) => ({ ...current, ...patch }));
    review.edited();
  };

  const apply = async () => {
    if (savingRef.current) return;
    savingRef.current = true;
    const count = reviewEntries.length;
    setConfirmMigration(false);
    setSaving(true);
    setError(undefined);
    review.started(count);

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
    savingRef.current = false;
    setSaving(false);
    // The circuit and its live region carry the outcome; no toast.
    review.settled(count, result);
    if (result.status === 'applied') {
      onApplied();
    } else {
      setError(t(applyErrorKey(result.error?.status)));
    }
  };

  // Save opens the review drawer; its Apply is the confirmation.
  const save = () => review.openReview();

  const applyFromReview = async () => {
    // The legacy migration keeps its own acknowledgement, after the review.
    if (isLegacySurface) {
      review.closeReview();
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
      circuit={{ circuit: review.circuit, pendingCount: reviewEntries.length }}
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
            className="input input-bordered mt-1 min-h-11 w-full md:min-h-0"
            data-testid={`boundary-${surface}-user`}
            value={value.user}
            onChange={(event) => edit({ user: event.target.value })}
          />
        </label>
        <label className="text-sm">
          {t(i18n('password'))}
          <input
            className="input input-bordered mt-1 min-h-11 w-full md:min-h-0"
            data-testid={`boundary-${surface}-pass`}
            type="password"
            value={value.pass}
            onChange={(event) => edit({ pass: event.target.value })}
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
            className="action-btn-primary min-h-11 md:min-h-0"
            onClick={save}
            disabled={!editable || saving}
            data-testid={`boundary-${surface}-save`}
          >
            {saving ? t(i18n('saving')) : t(i18n('save'))}
          </button>
        </div>
      </fieldset>
    </SecuritySurface>

      <ReviewChangesPanel
        isOpen={review.reviewOpen}
        entries={reviewEntries}
        pendingCount={reviewEntries.length}
        target="settings.js"
        isPending={saving}
        onApply={applyFromReview}
        onCancel={review.closeReview}
      />

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
