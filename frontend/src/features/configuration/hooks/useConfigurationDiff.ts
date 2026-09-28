import { useMemo } from 'react';
import type { HostStatus, NodeRedConfigFormData } from '@/shared/types';
import type { NodeRedConfigResponse } from '../lib/configTransformers';
import { configToFormData } from '../lib/configTransformers';
import { INITIAL_FORM_DATA } from '../lib/initialFormData';
import { validateConfigurationField } from '../lib/validateConfigurationField';

/**
 * Issue #766 slice F — six-element per-field header + safe-apply flow.
 *
 * `computeConfigurationDiff` is the single source of truth for the
 * six columns the operator needs in front of every form field:
 *
 *   configured value | effective value | source | validation |
 *   pending change   | restart required
 *
 * The `useConfigurationDiff` hook is a thin `useMemo` wrapper around it
 * so the component layer stays declarative. The pure function is
 * exported separately so tests and the future `<ReviewChangesPanel>`
 * share the exact same diff code.
 */

export type FieldSource = 'settings.js' | 'form default';

export type RestartSeverity = 'none' | 'soft' | 'hard';

export type ValidationState = 'valid' | 'pending' | 'invalid';

export interface FieldDiff {
  /** The value last persisted to settings.js (or the backend default). */
  configuredValue: unknown;
  /** The value NRCC has loaded right now from the backend. */
  effectiveValue: unknown;
  /** The value the operator has in the form right now. */
  formValue: unknown;
  /** Honest label for the effective column. */
  effectiveLabel: string;
  /** Where the field's persisted value comes from. */
  source: FieldSource;
  /** Whether formValue differs from configuredValue. */
  pending: boolean;
  /** Restart class the change triggers. */
  restart: RestartSeverity;
  /** Per-field validation. */
  validation: { state: ValidationState; message?: string };
}

export type ConfigurationDiff = Record<keyof NodeRedConfigFormData, FieldDiff>;

export interface ConfigurationDiffSummary {
  fields: ConfigurationDiff;
  pendingCount: number;
  restartRequired: RestartSeverity;
  validationErrors: Record<string, string>;
  canSave: boolean;
}

/**
 * Static map of which fields survive a `settings.js` reload. Fields not
 * listed are NRCC-side form defaults (e.g. UI affordances, write-only
 * passwords).
 */
const FIELD_SOURCE: Record<keyof NodeRedConfigFormData, FieldSource> = {
  // Basic
  uiPort: 'settings.js',
  uiHost: 'settings.js',
  httpAdminRoot: 'settings.js',
  httpNodeRoot: 'settings.js',
  disableEditor: 'settings.js',
  // Auth (write-only)
  authEnabled: 'settings.js',
  authAdminUser: 'settings.js',
  authAdminPassword: 'form default',
  authNodeHttpEnabled: 'settings.js',
  authNodeHttpUser: 'settings.js',
  authNodeHttpPassword: 'form default',
  authStaticEnabled: 'settings.js',
  authStaticUser: 'settings.js',
  authStaticPassword: 'form default',
  // Projects
  projectsEnabled: 'settings.js',
  // Logging
  loggingConsoleLevel: 'settings.js',
  loggingConsoleMetrics: 'settings.js',
  loggingInternalLevel: 'settings.js',
  loggingInternalMetrics: 'settings.js',
  // Files
  flowFile: 'settings.js',
  userDir: 'settings.js',
  nodesDir: 'settings.js',
  // Editor theme (runtime-reload)
  editorPageTitle: 'settings.js',
  editorPageFavicon: 'settings.js',
  editorPageCss: 'settings.js',
  editorHeaderTitle: 'settings.js',
  editorHeaderImage: 'settings.js',
  editorHeaderUrl: 'settings.js',
  editorDeployType: 'settings.js',
  editorDeployLabel: 'settings.js',
  editorDeployIcon: 'settings.js',
  editorPaletteEditable: 'settings.js',
  editorPaletteCatalogues: 'settings.js',
  editorProjectsEnabled: 'settings.js',
  editorCodeLib: 'settings.js',
  editorCodeTheme: 'settings.js',
  editorCodeFontSize: 'settings.js',
  editorUserMenu: 'settings.js',
  editorTours: 'settings.js',
  editorLoginImage: 'settings.js',
  editorLogoutRedirect: 'settings.js',
  // Runtime state
  runtimeStateEnabled: 'settings.js',
  runtimeStateFile: 'settings.js',
  // Language
  lang: 'settings.js',
  // Security slice (issue #762) — write-only secret, https fields.
  credentialSecret: 'form default',
  requireHttps: 'settings.js',
  httpsKey: 'settings.js',
  httpsCert: 'settings.js',
  httpsCA: 'settings.js',
  httpsPort: 'settings.js',
  httpsPassphrase: 'form default',
};

/**
 * Hard-restart fields: changing these requires Node-RED to be restarted
 * so the new value reaches the runtime.
 */
const HARD_RESTART_FIELDS: ReadonlySet<keyof NodeRedConfigFormData> = new Set([
  'uiPort',
  'uiHost',
  'httpAdminRoot',
  'httpNodeRoot',
  'disableEditor',
  'authEnabled',
  'authAdminUser',
  'authNodeHttpEnabled',
  'authNodeHttpUser',
  'authStaticEnabled',
  'authStaticUser',
  'flowFile',
  'userDir',
  'nodesDir',
  'editorCodeLib',
  'runtimeStateEnabled',
  'runtimeStateFile',
  'lang',
]);

/**
 * Soft-restart fields: changing these only needs a flow redeploy; no
 * settings.js reload.
 */
const SOFT_RESTART_FIELDS: ReadonlySet<keyof NodeRedConfigFormData> = new Set([
  'loggingConsoleLevel',
  'loggingConsoleMetrics',
  'loggingInternalLevel',
  'loggingInternalMetrics',
  'editorPageTitle',
  'editorPageFavicon',
  'editorPageCss',
  'editorHeaderTitle',
  'editorHeaderImage',
  'editorHeaderUrl',
  'editorDeployType',
  'editorDeployLabel',
  'editorDeployIcon',
  'editorPaletteEditable',
  'editorPaletteCatalogues',
  'editorProjectsEnabled',
  'editorCodeTheme',
  'editorCodeFontSize',
  'editorUserMenu',
  'editorTours',
  'editorLoginImage',
  'editorLogoutRedirect',
]);

function classifyRestart(key: keyof NodeRedConfigFormData): RestartSeverity {
  if (HARD_RESTART_FIELDS.has(key)) return 'hard';
  if (SOFT_RESTART_FIELDS.has(key)) return 'soft';
  return 'none';
}

function highestSeverity(left: RestartSeverity, right: RestartSeverity): RestartSeverity {
  const rank: Record<RestartSeverity, number> = { none: 0, soft: 1, hard: 2 };
  return rank[left] >= rank[right] ? left : right;
}

function equalValues(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  // String normalisation: trim trailing whitespace differences that
  // `configToFormData` would otherwise hide.
  if (typeof a === 'string' && typeof b === 'string') {
    return a.trim() === b.trim();
  }
  return false;
}

export function computeConfigurationDiff(
  formData: NodeRedConfigFormData,
  loadedConfig: NodeRedConfigResponse | null,
  hostStatus: HostStatus,
  rawSettingsContent: string
): ConfigurationDiffSummary {
  // configuredValue = effectiveValue on first render = what NRCC has
  // loaded. formValue is what the operator typed. The pending flag is
  // `formValue !== configuredValue`.
  const baseline: NodeRedConfigFormData = loadedConfig
    ? configToFormData(loadedConfig)
    : { ...INITIAL_FORM_DATA };

  const fields = {} as ConfigurationDiff;
  for (const key of Object.keys(baseline) as Array<keyof NodeRedConfigFormData>) {
    const configuredValue = baseline[key];
    const formValue = formData[key];
    const pending = !equalValues(formValue, configuredValue);
    const restart = classifyRestart(key);
    const validation = validateConfigurationField(key, formValue, rawSettingsContent);
    fields[key] = {
      configuredValue,
      effectiveValue: configuredValue,
      formValue,
      effectiveLabel: 'Effective (NRCC-loaded)',
      source: FIELD_SOURCE[key],
      pending,
      restart,
      validation,
    };
  }

  // Aggregate pending + restart
  let pendingCount = 0;
  let restartRequired: RestartSeverity = 'none';
  const validationErrors: Record<string, string> = {};
  for (const key of Object.keys(fields) as Array<keyof NodeRedConfigFormData>) {
    const f = fields[key];
    if (f.pending) pendingCount += 1;
    if (f.pending) restartRequired = highestSeverity(restartRequired, f.restart);
    if (f.validation.state === 'invalid' && f.validation.message) {
      validationErrors[key as string] = f.validation.message;
    }
  }

  // canSave requires pending changes AND no validation errors AND the
  // backend must not be in read-only mode.
  const canSave =
    pendingCount > 0 &&
    Object.keys(validationErrors).length === 0 &&
    hostStatus.configuration.editable !== false;

  // Reference hostStatus so the linter recognises it as part of the
  // diff inputs (used for read-only gating above).
  void hostStatus;

  return {
    fields,
    pendingCount,
    restartRequired,
    validationErrors,
    canSave,
  };
}

/**
 * Hook wrapper. Memoises on the four inputs so the result is stable
 * across renders unless something changed.
 */
export function useConfigurationDiff(
  formData: NodeRedConfigFormData,
  loadedConfig: NodeRedConfigResponse | null,
  hostStatus: HostStatus,
  rawSettingsContent: string
): ConfigurationDiffSummary {
  return useMemo(
    () => computeConfigurationDiff(formData, loadedConfig, hostStatus, rawSettingsContent),
    [formData, loadedConfig, hostStatus, rawSettingsContent]
  );
}