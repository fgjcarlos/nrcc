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

  authEnabled: false,
  authAdminUser: '',
  authAdminPassword: '',
  authNodeHttpEnabled: false,
  authNodeHttpUser: '',
  authNodeHttpPassword: '',
  authStaticEnabled: false,
  authStaticUser: '',
  authStaticPassword: '',

  projectsEnabled: false,

  loggingConsoleLevel: 'info',
  loggingConsoleMetrics: false,
  loggingInternalLevel: 'info',
  loggingInternalMetrics: false,

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
};