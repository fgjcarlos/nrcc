// TypeScript types for Node-RED full settings.js configuration
// Spec: Full settings.js Configuration UI — Phase 11

// ─────────────────── Server Section ───────────────────────

export type ServerConfig = {
  uiPort: number             // default: 1880
  uiHost: string             // default: "0.0.0.0"
  httpAdminRoot: string      // default: "/"
  httpNodeRoot: string       // default: "/"
  httpStatic: string         // default: ""
  disableEditor: boolean     // default: false
}

// ─────────────────── Security Section ───────────────────────

export type AdminAuthUser = {
  username: string
  password: string
  permissions: '*' | 'read'
}

export type AdminAuthDefault = {
  permissions: '*' | 'read'
}

export type AdminAuthConfig = {
  type: 'credentials' | 'strategy'
  users: AdminAuthUser[]
  default?: AdminAuthDefault
}

export type HTTPNodeAuthConfig = {
  user: string
  pass: string
}

export type SecurityConfig = {
  adminAuth?: AdminAuthConfig
  httpNodeAuth?: HTTPNodeAuthConfig
  credentialSecret: string
  sessionExpiryTime: number  // default: 86400 (seconds)
}

// ─────────────────── Editor Theme Section ───────────────────────

export type EditorPageConfig = {
  title: string
  favicon: string
  css: string
}

export type EditorHeaderConfig = {
  title: string
  image: string
  url: string
}

export type EditorDeployButtonConfig = {
  type: 'simple' | 'confirm'
  label: string
}

export type EditorCodeConfig = {
  lib: 'ace' | 'monaco'
  options: Record<string, string>
}

export type EditorThemeConfig = {
  theme: string
  page?: EditorPageConfig
  header?: EditorHeaderConfig
  deployButton?: EditorDeployButtonConfig
  tours: boolean             // default: true
  userMenu: boolean          // default: true
  projects: { enabled: boolean }
  codeEditor?: EditorCodeConfig
}

// ─────────────────── Flows Section ───────────────────────

export type FlowsConfig = {
  flowFile: string           // default: "flows.json"
  flowFilePretty: boolean    // default: false
  userDir: string            // default: ""
  nodesDir: string           // default: ""
}

// ─────────────────── Context Storage Section ───────────────────────

export type ContextStoreEntry = {
  module: 'memory' | 'localfilesystem'
  config?: Record<string, unknown>
}

export type ContextStorageConfig = {
  default: string
  stores: Record<string, ContextStoreEntry>
}

// ─────────────────── Logging Section ───────────────────────

export type ConsoleLogConfig = {
  level: 'fatal' | 'error' | 'warn' | 'info' | 'debug' | 'trace'
  metrics: boolean
  audit: boolean
}

export type LoggingConfig = {
  console: ConsoleLogConfig
}

// ─────────────────── Runtime Section ───────────────────────

export type ExternalModulesPaletteConfig = {
  allowInstall: boolean
  allowUpload: boolean
  allowList: string[]
  denyList: string[]
}

export type ExternalModulesModuleConfig = {
  allowInstall: boolean
  allowList: string[]
  denyList: string[]
}

export type ExternalModulesConfig = {
  autoInstall: boolean
  autoInstallRetry: number
  palette: ExternalModulesPaletteConfig
  modules: ExternalModulesModuleConfig
}

export type RuntimeConfig = {
  functionExternalModules: boolean
  functionTimeout: number
  debugMaxLength: number
  externalModules?: ExternalModulesConfig
  diagnosticsEnabled: boolean
  safeMode: boolean
  nodeMessageBufferMaxLength: number
}

// ─────────────────── HTTPS Section ───────────────────────

export type HTTPSConfig = {
  enabled: boolean
  keyFile: string
  certFile: string
  caFile: string
}

// ─────────────────── Node Reconnect Section ───────────────────────

export type NodeReconnectConfig = {
  mqttReconnectTime: number
  serialReconnectTime: number
  socketReconnectTime: number
  socketTimeout: number
}

// ─────────────────── Palette Section ───────────────────────

export type PaletteConfig = {
  categories: string[]
}

// ─────────────────── Full Config ───────────────────────

export type FullAppConfig = {
  server: ServerConfig
  security: SecurityConfig
  editorTheme: EditorThemeConfig
  flows: FlowsConfig
  contextStorage: ContextStorageConfig
  logging: LoggingConfig
  runtime: RuntimeConfig
  https: HTTPSConfig
  nodeReconnect: NodeReconnectConfig
  palette: PaletteConfig
}

// ─────────────────── Validation & API Types ───────────────────────

export type FieldError = {
  field: string   // dot-notation path, e.g., "server.uiPort"
  message: string
}

export type ConfigDiffEntry = {
  field: string
  oldValue: string
  newValue: string
}

export type ExtendedConfigValidationResult = {
  valid: boolean
  restartRequired: boolean
  errors: FieldError[]
  diff: ConfigDiffEntry[]
}

export type ConfigSnapshot = {
  id: string
  createdAt: string
  label: string
  reason: 'manual' | 'pre-apply' | 'pre-restore'
}

export type ConfigSnapshotList = {
  items: ConfigSnapshot[]
}

// ─────────────────── Defaults ───────────────────────

export function defaultFullAppConfig(): FullAppConfig {
  return {
    server: {
      uiPort: 1880,
      uiHost: '0.0.0.0',
      httpAdminRoot: '/',
      httpNodeRoot: '/',
      httpStatic: '',
      disableEditor: false,
    },
    security: {
      credentialSecret: '',
      sessionExpiryTime: 86400,
    },
    editorTheme: {
      theme: '',
      tours: true,
      userMenu: true,
      projects: { enabled: false },
    },
    flows: {
      flowFile: 'flows.json',
      flowFilePretty: false,
      userDir: '',
      nodesDir: '',
    },
    contextStorage: {
      default: 'default',
      stores: {
        default: {
          module: 'memory',
        },
      },
    },
    logging: {
      console: {
        level: 'info',
        metrics: false,
        audit: false,
      },
    },
    runtime: {
      functionExternalModules: false,
      functionTimeout: 0,
      debugMaxLength: 1000,
      diagnosticsEnabled: true,
      safeMode: false,
      nodeMessageBufferMaxLength: 0,
    },
    https: {
      enabled: false,
      keyFile: '',
      certFile: '',
      caFile: '',
    },
    nodeReconnect: {
      mqttReconnectTime: 5000,
      serialReconnectTime: 5000,
      socketReconnectTime: 10000,
      socketTimeout: 120000,
    },
    palette: {
      categories: ['subflows', 'common', 'function', 'network', 'sequence', 'parser', 'storage'],
    },
  }
}
