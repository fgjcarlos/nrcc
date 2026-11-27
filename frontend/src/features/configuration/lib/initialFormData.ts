import { DEFAULT_HTTP_NODE_CORS_METHODS } from '@/shared/types';
import type { NodeRedConfigFormData } from '@/shared/types';

/**
 * Single source of truth for the empty-state form used by both
 * `ConfigurationView` and the slice F diff/validation tests. Extracted
 * from `ConfigurationView.tsx` so test fixtures stay in sync with the
 * component initial state.
 *
 * Issue #766 slice F — Configuration six-element header + safe-apply.
 */
export const INITIAL_FORM_DATA: NodeRedConfigFormData = {
  uiPort: 1880,
  uiHost: '0.0.0.0',
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  disableEditor: false,


  projectsEnabled: false,

  loggingConsoleLevel: 'info',
  loggingConsoleMetrics: false,
  loggingConsoleAudit: undefined,
  loggingInternalLevel: 'info',
  loggingInternalMetrics: false,
  loggingInternalAudit: undefined,

  flowFile: 'flows.json',
  userDir: '',
  nodesDir: '',

  editorPageTitle: 'Node-RED',
  editorPageFavicon: '',
  editorPageCss: '',
  editorHeaderTitle: 'Node-RED',
  editorHeaderImage: '',
  editorHeaderUrl: '',
  editorDeployType: 'default',
  editorDeployLabel: 'Deploy',
  editorDeployIcon: '',
  editorPaletteEditable: true,
  editorPaletteCatalogues: '',
  editorProjectsEnabled: false,
  editorCodeLib: 'ace',
  editorCodeTheme: 'vs',
  editorCodeFontSize: 12,
  editorUserMenu: true,
  editorTours: true,
  editorLoginImage: '',
  editorLogoutRedirect: '',

  runtimeStateEnabled: false,
  runtimeStateFile: '',

  lang: 'en-US',

  // CORS (issue #1076) — mirrors the defaults `configToFormData` produces
  // for an absent `httpNodeCors` key.
  httpNodeCorsEnabled: false,
  httpNodeCorsOrigins: '',
  httpNodeCorsMethods: DEFAULT_HTTP_NODE_CORS_METHODS.join(','),
  httpNodeCorsCredentials: false,
  httpNodeCorsAllowedHeaders: '',
  httpNodeCorsUnsupported: false,
  httpNodeCorsRaw: '',
};