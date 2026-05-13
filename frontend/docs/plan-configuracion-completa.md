# Plan: Configuración Completa de Node-RED

## Objetivo
Permitir modificar **TODAS** las opciones de configuración que soporta Node-RED en `settings.js` desde una interfaz web completa.

---

## 1. Opciones de Configuración Node-RED

### 1.1 Configuración Básica (Basic Settings)

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `uiPort` | number | 1880 | Puerto del editor |
| `uiHost` | string | "0.0.0.0" | Interface de escucha |
| `httpAdminRoot` | string | "/" | URL raíz del editor |
| `httpNodeRoot` | string | "/" | URL raíz de nodos HTTP |
| `httpRoot` | string | - | Override para ambos |
| `disableEditor` | boolean | false | Deshabilitar editor |

### 1.2 Proyectos (Projects)

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `projects.enabled` | boolean | false | Habilitar proyectos |

### 1.3 Logging

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `logging.level` | string | "info" | Nivel de log |
| `logging.console.level` | string | "info" | Nivel de log en consola |

### 1.4 Archivos (Files)

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `flowFile` | string | "flows_<hostname>.json" | Archivo de flows |
| `userDir` | string | "$HOME/.node-red" | Directorio de usuario |
| `nodesDir` | string | - | Directorio adicional de nodos |

### 1.5 Configuración del Editor (Editor Theme)

#### Page
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.page.title` | string | Título de la página |
| `editorTheme.page.favicon` | string | Ruta al favicon |
| `editorTheme.page.css` | string | Ruta a CSS custom |
| `editorTheme.page.scripts` | array | Scripts custom |

#### Header
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.header.title` | string | Título en header |
| `editorTheme.header.image` | string | Imagen en header |
| `editorTheme.header.url` | string | Link del header |

#### Deploy Button
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.deployButton.type` | string | Tipo: "default", "simple" |
| `editorTheme.deployButton.label` | string | Texto del botón |
| `editorTheme.deployButton.icon` | string | Icono del botón |

#### Menu
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.menu` | object | Items del menú (mostrar/ocultar) |

#### Palette
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.palette.editable` | boolean | Permitir instalación de nodos |
| `editorTheme.palette.catalogues` | array | Catálogos de nodos |
| `editorTheme.palette.theme` | array | Colores de nodos |

#### Projects
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.projects.enabled` | boolean | Habilitar proyectos |

#### Code Editor
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.codeEditor.lib` | string | "ace" o "monaco" |
| `editorTheme.codeEditor.options` | object | Opciones del editor |

#### Theme
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.theme` | string | Tema del editor |

#### User Menu
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.userMenu` | boolean | Mostrar menú de usuario |

#### Tours
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.tours` | boolean | Habilitar tours de bienvenida |

#### Login
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.login.image` | string | Imagen de login |

#### Logout
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `editorTheme.logout.redirect` | string | URL de redirect al logout |

### 1.6 Runtime State

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `runtimeState.enabled` | boolean | false | Habilitar estado del runtime |
| `runtimeState.file` | string | - | Archivo de estado |

### 1.7 Idioma

| Campo | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `lang` | string | "en-US" | Idioma del runtime |

---

## 2. Estructura de la Página de Configuración

```
┌─────────────────────────────────────────────────────────────────────┐
│  Node-RED Configuration                              [Save] [Reset]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────┐  ┌────────────────────────────────────────┐│
│  │  BASIC SETTINGS  │  │  Port: [____]                          ││
│  │  ──────────────  │  │  Host: [________________]              ││
│  │  □ Projects      │  │  Admin Root: [________________]        ││
│  │  ▶ Editor       │  │  Disable Editor: [toggle]              ││
│  │  ▶ Logging      │  │                                        ││
│  │  ▶ Files        │  └────────────────────────────────────────┘│
│  │  ▶ Runtime      │                                              │
│  │  ▶ Security     │                                              │
│  └──────────────────┘                                              │
│                                                                     │
│  o sections with expandable fields...                              │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Tipos TypeScript Completos

```typescript
// ============================================
// CONFIGURATION TYPES
// ============================================

// Basic Settings
export interface BasicSettings {
  uiPort: number;
  uiHost?: string;
  httpAdminRoot?: string | false;
  httpNodeRoot?: string | false;
  httpRoot?: string;
  disableEditor?: boolean;
}

// Projects
export interface ProjectSettings {
  projectsEnabled: boolean;
}

// Logging
export interface LoggingSettings {
  loggingLevel: 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  consoleLevel?: 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';
}

// Files
export interface FileSettings {
  flowFile?: string;
  userDir?: string;
  nodesDir?: string;
}

// Editor Theme - Page
export interface EditorPage {
  title?: string;
  favicon?: string;
  css?: string | string[];
  scripts?: string | string[];
}

// Editor Theme - Header
export interface EditorHeader {
  title?: string;
  image?: string;
  url?: string;
}

// Editor Theme - Deploy Button
export interface EditorDeployButton {
  type?: 'default' | 'simple' | 'icon';
  label?: string;
  icon?: string;
}

// Editor Theme - Palette Catalogue
export interface EditorPaletteCatalogue {
  id: string;
  url: string;
  label?: string;
}

// Editor Theme - Palette Node Color
export interface EditorPaletteTheme {
  category?: string;
  type?: string;
  color?: string;
}

// Editor Theme - Palette
export interface EditorPalette {
  editable?: boolean;
  catalogues?: string[];
  theme?: EditorPaletteTheme[];
}

// Editor Theme - Projects
export interface EditorProjects {
  enabled?: boolean;
}

// Editor Theme - Code Editor
export interface EditorCodeEditorOptions {
  theme?: string;
  fontSize?: number;
  fontFamily?: string;
  tabSize?: number;
  minimap?: boolean;
  lineNumbers?: boolean;
  foldGutter?: boolean;
  wordWrap?: boolean;
}

export interface EditorCodeEditor {
  lib?: 'ace' | 'monaco';
  options?: EditorCodeEditorOptions;
}

// Editor Theme - Menu Items
export interface EditorMenuItem {
  label?: string;
  url?: string;
}

export interface EditorMenu {
  'menu-item-import-library'?: boolean | EditorMenuItem;
  'menu-item-export-library'?: boolean | EditorMenuItem;
  'menu-item-keyboard-shortcuts'?: boolean;
  'menu-item-help'?: boolean | EditorMenuItem;
  'menu-item-welcome'?: boolean;
  'menu-item-nodes'?: boolean;
  'menu-item-view'?: boolean;
  'menu-item-users'?: boolean;
  'menu-item-settings'?: boolean;
  'menu-item-install'?: boolean;
  'menu-item-project'?: boolean;
  'menu-item-subflow'?: boolean;
  'menu-item-examples'?: boolean;
  [key: string]: boolean | EditorMenuItem | undefined;
}

// Editor Theme - Login
export interface EditorLogin {
  image?: string;
}

// Editor Theme - Logout
export interface EditorLogout {
  redirect?: string;
}

// Editor Theme - Full
export interface EditorTheme {
  page?: EditorPage;
  header?: EditorHeader;
  deployButton?: EditorDeployButton;
  menu?: EditorMenu;
  palette?: EditorPalette;
  projects?: EditorProjects;
  codeEditor?: EditorCodeEditor;
  theme?: string;
  userMenu?: boolean;
  tours?: boolean;
  login?: EditorLogin;
  logout?: EditorLogout;
  mermaid?: {
    theme?: 'default' | 'base' | 'forest' | 'dark' | 'neutral';
  };
}

// Runtime State
export interface RuntimeStateSettings {
  enabled?: boolean;
  file?: string;
}

// Language
export interface LanguageSettings {
  lang?: string;
}

// Complete Config
export interface NodeRedConfig {
  // Basic
  uiPort: number;
  uiHost?: string;
  httpAdminRoot?: string | false;
  httpNodeRoot?: string | false;
  httpRoot?: string;
  disableEditor?: boolean;
  
  // Projects
  projectsEnabled: boolean;
  
  // Logging
  loggingLevel: 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  
  // Files
  flowFile?: string;
  userDir?: string;
  nodesDir?: string;
  
  // Editor
  editorTheme?: EditorTheme;
  
  // Runtime
  runtimeState?: RuntimeStateSettings;
  
  // Language
  lang?: string;
}
```

---

## 4. Componentes a Crear

### frontend/src/components/config/

```
components/config/
├── ConfigPage.tsx           # Página principal
├── ConfigForm.tsx           # Formulario completo
├── ConfigSidebar.tsx        # Navegación por secciones
├── sections/
│   ├── BasicSection.tsx     # Configuración básica
│   ├── ProjectsSection.tsx  # Proyectos
│   ├── LoggingSection.tsx   # Logging
│   ├── FilesSection.tsx     # Archivos
│   ├── EditorSection.tsx    # Tema del editor
│   ├── RuntimeSection.tsx   # Runtime state
│   └── LanguageSection.tsx  # Idioma
├── ConfigPreview.tsx        # Preview JSON
├── ConfigActions.tsx        # Botones Save/Reset
└── types.ts                 # Tipos específicos
```

---

## 5. Implementación por Fases

### Fase 1: Estructura Base
- [ ] Crear tipos completos en `types.ts`
- [ ] Crear ConfigPage.tsx
- [ ] Crear ConfigSidebar.tsx con navegación

### Fase 2: Secciones del Formulario
- [ ] BasicSection (uiPort, uiHost, httpAdminRoot, etc.)
- [ ] ProjectsSection
- [ ] LoggingSection
- [ ] FilesSection

### Fase 3: Editor Theme
- [ ] EditorSection completa
- [ ] Sub-secciones: Page, Header, Deploy, Menu, Palette, CodeEditor

### Fase 4: Runtime y Language
- [ ] RuntimeSection
- [ ] LanguageSection

### Fase 5: Preview y Guardado
- [ ] ConfigPreview.tsx
- [ ] Conexión con API
- [ ] Validación
- [ ] Toast notifications

---

## 6. UI Mockup Final

```
┌────────────────────────────────────────────────────────────────────────┐
│  Node-RED Configuration                              [Save] [Reset]   │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌─────────────┐ ┌──────────────────────────────────────────────────┐  │
│  │ BASIC       │ │ ┌──────────────────────────────────────────────┐ │  │
│  │ ─────────── │ │ │ UI Settings                                  │ │  │
│  │ Projects    │ │ │                                              │ │  │
│  │ Logging     │ │ │ Port: [1880]                                 │ │  │
│  │ Files       │ │ │ Host: [0.0.0.0________________]              │ │  │
│  │ Editor      │ │ │ Admin Root: [/_____]  [toggle: Disable Editor]│ │  │
│  │ Runtime     │ │ └──────────────────────────────────────────────┘ │  │
│  │ Language    │ │ ┌──────────────────────────────────────────────┐ │  │
│  │             │ │ │ Projects                                     │ │  │
│  │             │ │ │                                              │ │  │
│  │             │ │ │ [✓] Enable Projects                         │ │  │
│  │             │ │ └──────────────────────────────────────────────┘ │  │
│  └─────────────┘ └──────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │ JSON Preview                                    [▼] [▲]      │       │
│  │ {                                                              │       │
│  │   "uiPort": 1880,                                           │       │
│  │   "projectsEnabled": false,                                  │       │
│  │   "loggingLevel": "info"                                     │       │
│  │ }                                                            │       │
│  └──────────────────────────────────────────────────────────────┘       │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Servicios API

```typescript
// GET /api/config
// POST /api/config
// POST /api/config/validate
```

---

## 8. Validación Zod (Backend)

```typescript
export const NodeRedConfigSchema = z.object({
  // Basic
  uiPort: z.number().min(1).max(65535).default(1880),
  uiHost: z.string().optional(),
  httpAdminRoot: z.union([z.string(), z.boolean()]).optional(),
  httpNodeRoot: z.union([z.string(), z.boolean()]).optional(),
  httpRoot: z.string().optional(),
  disableEditor: z.boolean().optional(),
  
  // Projects
  projectsEnabled: z.boolean().default(false),
  
  // Logging
  loggingLevel: z.enum(['trace', 'debug', 'info', 'warn', 'error', 'fatal']).default('info'),
  
  // Files
  flowFile: z.string().optional(),
  userDir: z.string().optional(),
  nodesDir: z.string().optional(),
  
  // Editor Theme (complejo - objeto anidado)
  editorTheme: z.object({
    page: z.object({
      title: z.string().optional(),
      favicon: z.string().optional(),
      css: z.union([z.string(), z.array(z.string())]).optional(),
      scripts: z.union([z.string(), z.array(z.string())]).optional(),
    }).optional(),
    // ... más campos
  }).optional(),
  
  // Runtime
  runtimeState: z.object({
    enabled: z.boolean().optional(),
    file: z.string().optional(),
  }).optional(),
  
  // Language
  lang: z.string().optional(),
});
```

---

## 9. Orden de Implementación

### Semana 1
1. Completar tipos TypeScript
2. Crear estructura de componentes
3. Implementar BasicSection

### Semana 2
4. ProjectsSection
5. LoggingSection
6. FilesSection

### Semana 3
7. EditorSection (completa)
8. RuntimeSection
9. LanguageSection

### Semana 4
10. ConfigPreview
11. Conexión API
12. Validación
13. Testing

---

## 10. Próximo Paso

Comenzar con **Fase 1**:
1. Completar tipos en frontend/src/types/index.ts
2. Crear estructura de componentes
3. Implementar BasicSection
