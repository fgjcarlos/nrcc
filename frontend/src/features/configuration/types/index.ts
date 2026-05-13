// Re-export configuration-specific types from shared
// These were originally in shared/types but are only used by configuration
export type {
  NodeRedConfigFormData,
  LoggingLevel,
  NodeRedConfig,
  LoggingSettings,
  EditorTheme,
  EditorPage,
  EditorHeader,
  EditorDeployButton,
  EditorMenuItem,
  EditorMenu,
  EditorPaletteCatalogue,
  EditorPaletteThemeItem,
  EditorPalette,
  EditorProjects,
  EditorCodeEditorOptions,
  EditorCodeEditor,
  EditorLogin,
  EditorLogout,
  EditorMermaid,
  AdminAuthSettings,
  NodeHttpAuthSettings,
  StaticAuthSettings,
  AuthSettings,
  RuntimeStateSettings,
} from '@/shared/types';

export { configToFormData, formDataToConfig, DEFAULT_CONFIG } from '@/shared/types';
