# Plan de Actuación - Frontend (React UI)

## 1. Análisis del Servicio

### Propósito
Panel administrativo web para gestionar Node-RED:
- Dashboard con estado del sistema
- Editor visual de configuración
- Monitor de logs en tiempo real
- Controles de runtime y Docker
- Diseño responsive y accesible

### Tecnologías
- React 18+ (Vite como bundler)
- React Router para navegación
- TanStack Query para estado servidor
- Zustand para estado cliente
- Tailwind CSS para estilos
- Shadcn/UI para componentes base

### Puertos
- `VITE_API_URL`: URL de la API (default: http://localhost:3000/api)

---

## 2. Dependencias

### Producción
```json
{
  "react": "^18.x",
  "react-dom": "^18.x",
  "react-router-dom": "^6.x",
  "@tanstack/react-query": "^5.x",
  "zustand": "^4.x",
  "axios": "^1.x",
  "tailwindcss": "^3.x",
  "clsx": "^2.x",
  "class-variance-authority": "^0.7.x",
  "lucide-react": "^0.x",
  "date-fns": "^3.x",
  "react-hook-form": "^7.x",
  "@hookform/resolvers": "^3.x",
  "zod": "^3.x",
  "sonner": "^1.x"
}
```

### Desarrollo
```json
{
  "vite": "^5.x",
  "@types/react": "^18.x",
  "@types/react-dom": "^18.x",
  "eslint": "^8.x",
  "prettier": "^3.x",
  "@typescript-eslint/eslint-plugin": "^6.x",
  "@typescript-eslint/parser": "^6.x"
}
```

---

## 3. Estructura de Archivos Propuesta

```
frontend/
├── src/
│   ├── components/
│   │   ├── ui/              # Componentes base (shadcn)
│   │   │   ├── Button.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Switch.tsx
│   │   │   ├── Select.tsx
│   │   │   ├── Dialog.tsx
│   │   │   ├── Tabs.tsx
│   │   │   ├── Badge.tsx
│   │   │   ├── Skeleton.tsx
│   │   │   ├── ScrollArea.tsx
│   │   │   └── ...
│   │   │
│   │   ├── layout/          # Componentes de layout
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   ├── Layout.tsx
│   │   │   └── MobileNav.tsx
│   │   │
│   │   ├── dashboard/       # Componentes del dashboard
│   │   │   ├── StatusCard.tsx
│   │   │   ├── SystemInfo.tsx
│   │   │   ├── QuickActions.tsx
│   │   │   ├── ActivityFeed.tsx
│   │   │   └── UptimeDisplay.tsx
│   │   │
│   │   ├── config/          # Componentes de configuración
│   │   │   ├── ConfigForm.tsx
│   │   │   ├── ConfigEditor.tsx
│   │   │   ├── ConfigPreview.tsx
│   │   │   ├── ConfigField.tsx
│   │   │   └── ConfigActions.tsx
│   │   │
│   │   ├── runtime/         # Componentes de runtime
│   │   │   ├── RuntimeStatus.tsx
│   │   │   ├── RestartButton.tsx
│   │   │   ├── ProcessInfo.tsx
│   │   │   └── UptimeCounter.tsx
│   │   │
│   │   ├── logs/            # Componentes de logs
│   │   │   ├── LogViewer.tsx
│   │   │   ├── LogFilter.tsx
│   │   │   ├── LogEntry.tsx
│   │   │   ├── LogControls.tsx
│   │   │   └── LogSearch.tsx
│   │   │
│   │   ├── docker/          # Componentes de docker
│   │   │   ├── ContainerStatus.tsx
│   │   │   ├── ContainerInfo.tsx
│   │   │   ├── DockerActions.tsx
│   │   │   └── PortMappings.tsx
│   │   │
│   │   └── common/          # Componentes reutilizables
│   │       ├── Loading.tsx
│   │       ├── Error.tsx
│   │       ├── ConfirmDialog.tsx
│   │       ├── PageHeader.tsx
│   │       └── EmptyState.tsx
│   │
│   ├── pages/
│   │   ├── Dashboard.tsx
│   │   ├── Configuration.tsx
│   │   ├── Runtime.tsx
│   │   ├── Logs.tsx
│   │   ├── Docker.tsx
│   │   └── Settings.tsx
│   │
│   ├── services/            # Comunicación con API
│   │   ├── api.ts           # Axios instance
│   │   ├── configService.ts
│   │   ├── runtimeService.ts
│   │   ├── dockerService.ts
│   │   ├── systemService.ts
│   │   └── logService.ts
│   │
│   ├── types/               # Tipos TypeScript
│   │   ├── config.ts
│   │   ├── runtime.ts
│   │   ├── docker.ts
│   │   ├── system.ts
│   │   └── api.ts
│   │
│   ├── stores/              # Estado global (Zustand)
│   │   ├── useConfigStore.ts
│   │   ├── useRuntimeStore.ts
│   │   └── useUIStore.ts
│   │
│   ├── hooks/               # Custom hooks
│   │   ├── useConfig.ts
│   │   ├── useRuntime.ts
│   │   ├── useLogs.ts
│   │   ├── usePolling.ts
│   │   └── useDebounce.ts
│   │
│   ├── lib/                 # Utilidades
│   │   ├── utils.ts
│   │   ├── constants.ts
│   │   └── formatters.ts
│   │
│   ├── App.tsx              # Root component
│   ├── main.tsx             # Entry point
│   └── index.css            # Estilos globales
│
├── public/
│   └── favicon.ico
│
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
├── tailwind.config.js
└── postcss.config.js
```

---

## 4. Detalle de Páginas

### Dashboard (`/dashboard`)

**Propósito**: Vista principal que muestra el estado general del sistema.

**Componentes principales:**

1. **StatusCard** (múltiples)
   - Estado de Node-RED (running/stopped/error)
   - Estado del contenedor Docker
   - Uso de CPU
   - Uso de memoria

2. **QuickActions**
   - Botón de restart rápido
   - Botón de reload config
   - Botón de view logs

3. **UptimeDisplay**
   - Tiempo desde último restart
   - Formato: "X días, Y horas, Z minutos"

4. **SystemInfo**
   - CPU usage (porcentaje + gráfico)
   - Memoria (usada/total)
   - Disco (usado/total)
   - Uptime del sistema host

5. **ActivityFeed**
   - Lista de eventos recientes
   - Tipo: info, warning, error
   - Timestamp relativo ("hace 2 minutos")

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  Header: "Dashboard" + Quick Actions                 │
├─────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │ Node-RED │  │Container │  │   CPU    │         │
│  │  Status  │  │  Status  │  │  Usage   │         │
│  └──────────┘  └──────────┘  └──────────┘         │
├─────────────────────────────────────────────────────┤
│  ┌──────────────────────┐  ┌───────────────────┐  │
│  │    System Info        │  │   Uptime          │  │
│  │    (RAM, Disk)        │  │   Display         │  │
│  └──────────────────────┘  └───────────────────┘  │
├─────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────┐ │
│  │           Activity Feed                       │ │
│  │  - Runtime restarted (hace 5 min)            │ │
│  │  - Config updated (hace 1 hora)             │ │
│  │  - Docker connected (hace 2 horas)          │ │
│  └──────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**Comportamiento:**
- Polling cada 5 segundos para status
- Auto-refresh al cambiar de pestaña
- Animación suave en actualizaciones

---

### Configuración (`/settings`)

**Propósito**: Editor visual de configuración de Node-RED.

**Campos del formulario:**

1. **Basic Settings**
   - `uiPort`: Puerto UI (number, 1-65535)
   - `projectsEnabled`: Habilitar proyectos (switch)
   - `loggingLevel`: Nivel de log (select: trace/debug/info/warn/error/fatal)

2. **File Settings**
   - `flowFile`: Archivo de flows (text)
   - `userDir`: Directorio de usuario (text)
   - `nodesDir`: Directorio de nodos (text)

3. **Editor Theme**
   - `projects.enabled`: Habilitar proyectos en editor (switch)
   - `palette.catalogue`: Array de catálogos de nodos (tags input)
   - `menu.items`: Elementos del menú (multi-select)
   - `userMenu`: Mostrar menú de usuario (switch)

4. **Runtime State**
   - `runtimeState.enabled`: Habilitar estado de runtime (switch)
   - `runtimeState.file`: Archivo de estado (text)

**Componentes principales:**

1. **ConfigForm**
   - Formulario con react-hook-form
   - Validación con Zod
   - Agrupación en tabs/secciones
   - Dirty state tracking

2. **ConfigPreview**
   - JSON view de la configuración actual
   - Syntax highlighting
   - Copy to clipboard

3. **ConfigActions**
   - Save button (disabled si no hay cambios)
   - Reset button (confirmation dialog)
   - Apply & Restart button

**Estados:**
- Loading: skeleton loaders
- Editing: campos editables
- Saving: loading state en botón
- Success: toast notification
- Error: inline error messages

**Validaciones:**
- Puerto: rango válido
- Paths: validación de formato
- JSON: validación de estructura

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  Header: "Configuration" + Actions (Save/Reset)     │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐│
│  │  [Basic] [Files] [Theme] [Runtime]             ││
│  ├─────────────────────────────────────────────────┤│
│  │                                                 ││
│  │  Form Fields...                                ││
│  │                                                 ││
│  │  ┌─────────────────────────────────────────┐   ││
│  │  │ Port: [____]                            │   ││
│  │  │ Projects: [toggle]                       │   ││
│  │  │ Log Level: [dropdown]                    │   ││
│  │  └─────────────────────────────────────────┘   ││
│  │                                                 ││
│  └─────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────┤
│  Preview (JSON):                                    │
│  ┌─────────────────────────────────────────────────┐│
│  │ { "uiPort": 1880, ... }                        ││
│  └─────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────┘
```

---

### Runtime (`/runtime`)

**Propósito**: Control y monitoreo del runtime de Node-RED.

**Componentes principales:**

1. **RuntimeStatus**
   - Indicador visual grande
   - Texto: "Running" / "Stopped" / "Error"
   - Color: verde / rojo / amarillo

2. **UptimeCounter**
   - Tiempo activo en formato legible
   - Actualización en tiempo real
   - Formato: "2d 14h 32m 15s"

3. **ProcessInfo**
   - PID del proceso
   - Memoria RSS
   - Memoria Heap
   - Versión de Node-RED

4. **RestartButton**
   - Botón principal prominent
   - Estados: idle, restarting, loading
   - Diálogo de confirmación
   - Loading spinner durante restart
   - Toast al completar

**Acciones:**
- Restart Runtime: reinicia el proceso Node-RED
- View Logs: navega a página de logs

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  Header: "Runtime"                                  │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐│
│  │                                                 ││
│  │         ●  RUNNING                              ││
│  │      Uptime: 2d 14h 32m                        ││
│  │                                                 ││
│  │    [  RESTART RUNTIME  ]                       ││
│  │                                                 ││
│  └─────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────┤
│  Process Info:                                      │
│  ┌─────────────┬─────────────┬─────────────┐        │
│  │    PID      │   Memory    │   Version   │        │
│  │   12345     │   128 MB    │   3.1.0     │        │
│  └─────────────┴─────────────┴─────────────┘        │
└─────────────────────────────────────────────────────┘
```

**Comportamiento:**
- Polling cada 5 segundos
- Optimistic UI en restart
- Redirect a logs después de restart

---

### Logs (`/logs`)

**Propósito**: Monitor de logs en tiempo real de Node-RED.

**Componentes principales:**

1. **LogViewer**
   - Consola/terminal estilo
   - Scrolling automático (toggleable)
   - Colores por nivel:
     - INFO: blanco
     - WARN: amarillo
     - ERROR: rojo
     - DEBUG: gris
   - Timestamps
   - Virtual scrolling para performance

2. **LogControls**
   - Play/Pause streaming
   - Clear logs
   - Download logs (.txt)
   - Auto-scroll toggle

3. **LogFilter**
   - Filter por nivel (checkboxes)
   - Search/filter por texto
   - Date range (opcional)

4. **LogSearch**
   - Input de búsqueda
   - Highlight de matches
   - Regex toggle (opcional)

**Datos del log:**
```typescript
interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  source?: string;
}
```

**Características técnicas:**
- SSE connection para streaming
- Buffer local de 1000 entradas máximo
- Pause/Resume sin perder datos
- Reconexión automática en disconnect

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  Header: "Logs" + Controls                          │
│  [Play/Pause] [Clear] [Download] [Auto-scroll]    │
├─────────────────────────────────────────────────────┤
│  Filters: [v] Info [v] Warn [v] Error    [____] │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐│
│  │ 2024-01-15 10:23:45 [INFO] Node-RED started    ││
│  │ 2024-01-15 10:23:46 [INFO] Flows loaded: 12   ││
│  │ 2024-01-15 10:24:01 [WARN] Missing node: x    ││
│  │ 2024-01-15 10:25:33 [ERROR] Connection lost   ││
│  │ ...                                            ││
│  └─────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────┘
```

**Comportamiento:**
- Conexión SSE al montar
- Reconexión en caso de error
- Scroll to bottom automático (si enabled)
- Pause no cierra conexión, solo retiene

---

### Docker (`/docker`)

**Propósito**: Información y control del contenedor Docker.

**Componentes principales:**

1. **ContainerStatus**
   - Estado visual (running/stopped/paused)
   - Badge de estado
   - Tiempo corriendo

2. **ContainerInfo**
   - Container ID (truncado)
   - Image name
   - Created date
   - Networks

3. **PortMappings**
   - Lista de puertos expuestos
   - Formato: public → private (protocol)

4. **DockerActions**
   - Restart container
   - Stop container
   - View container logs (navega a /logs)

5. **ResourceUsage**
   - CPU percentage
   - Memory usage
   - Network I/O

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│  Header: "Docker" + Actions                         │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐│
│  │  Container Status                               ││
│  │  ● Running                                      ││
│  │  Image: nodered/node-red:3.1                    ││
│  │  Created: Jan 15, 2024                          ││
│  │                                                 ││
│  │  [RESTART CONTAINER]  [STOP]                   ││
│  └─────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────┤
│  Port Mappings:                                     │
│  ┌─────────────────────────────────────────────────┐│
│  │  1880 → 1880 (TCP)  Node-RED UI                ││
│  │  3000 → 3000 (TCP)  Control Center            ││
│  └─────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────┤
│  Resources:                                        │
│  CPU: ████████░░ 80%                               │
│  Memory: ██████░░░░ 60%                           │
└─────────────────────────────────────────────────────┘
```

---

### Settings (`/settings-app`)

**Propósito**: Configuración de la propia aplicación Control Center.

**Opciones:**
- Theme (light/dark/system)
- Polling interval
- Log buffer size
- Notifications (enable/disable)
- API timeout

---

## 5. Tipos TypeScript

```typescript
// types/config.ts
export interface NodeRedConfig {
  uiPort: number;
  projectsEnabled: boolean;
  loggingLevel: 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  flowFile?: string;
  userDir?: string;
  nodesDir?: string;
  editorTheme?: {
    projects?: { enabled?: boolean };
    palette?: { catalogue?: string[] };
    menu?: { items?: string[] };
    userMenu?: boolean;
  };
  runtimeState?: {
    enabled?: boolean;
    file?: string;
  };
}

// types/runtime.ts
export interface RuntimeInfo {
  status: 'running' | 'stopped' | 'error' | 'unknown';
  pid?: number;
  uptime: number;
  memory?: {
    rss: number;
    heapTotal: number;
    heapUsed: number;
  };
  version?: string;
}

// types/docker.ts
export interface ContainerInfo {
  id: string;
  name: string;
  image: string;
  status: 'running' | 'exited' | 'paused' | 'created' | 'restarting';
  created: string;
  ports: { privatePort: number; publicPort?: number; type: string }[];
  state: {
    running: boolean;
    paused: boolean;
    memory: number;
    cpu: number;
  };
}

// types/api.ts
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}
```

---

## 6. Servicios API

```typescript
// services/api.ts
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 10000,
});

// Interceptors para manejo de errores
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Redirect to login
    }
    return Promise.reject(error);
  }
);

// services/configService.ts
export const configService = {
  getConfig: () => api.get<NodeRedConfig>('/config'),
  updateConfig: (config: Partial<NodeRedConfig>) => 
    api.post('/config', config),
  validateConfig: (config: Partial<NodeRedConfig>) => 
    api.post('/config/validate', config),
};

// services/runtimeService.ts
export const runtimeService = {
  getStatus: () => api.get<RuntimeInfo>('/runtime/status'),
  restart: () => api.post<void>('/runtime/restart'),
  getUptime: () => api.get<{ uptime: number }>('/runtime/uptime'),
};

// services/logService.ts
export const logService = {
  getLogs: (lines?: number) => api.get<LogEntry[]>('/runtime/logs', {
    params: { lines }
  }),
  streamLogs: () => new EventSource(`${api.defaults.baseURL}/runtime/logs/stream`),
};
```

---

## 7. Stores (Zustand)

```typescript
// stores/useUIStore.ts
interface UIState {
  theme: 'light' | 'dark' | 'system';
  sidebarOpen: boolean;
  toasts: Toast[];
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  toggleSidebar: () => void;
  addToast: (toast: Omit<Toast, 'id'>) => void;
  removeToast: (id: string) => void;
}

// stores/useConfigStore.ts
interface ConfigState {
  localConfig: NodeRedConfig | null;
  isDirty: boolean;
  setLocalConfig: (config: NodeRedConfig) => void;
  updateField: (path: string, value: unknown) => void;
  reset: () => void;
}
```

---

## 8. TanStack Query Keys

```typescript
export const queryKeys = {
  config: ['config'] as const,
  runtimeStatus: ['runtime', 'status'] as const,
  runtimeUptime: ['runtime', 'uptime'] as const,
  containerStatus: ['docker', 'status'] as const,
  containerInfo: ['docker', 'info'] as const,
  systemInfo: ['system', 'info'] as const,
  logs: (lines?: number) => ['logs', lines] as const,
};
```

---

## 9. Orden de Implementación

### Fase 1: Setup (Semana 1)
1. **Inicializar proyecto Vite + TypeScript**
2. **Instalar dependencias**
3. **Configurar Tailwind CSS**
4. **Configurar Shadcn/UI**
5. **Crear tipos TypeScript**
6. **API client con Axios**

### Fase 2: Core UI (Semana 2)
7. **Router configurado**
8. **Layout base (Sidebar, Header)**
9. **Componentes base (Button, Card, Input, etc.)**
10. **Toast notifications (sonner)**

### Fase 3: Dashboard (Semana 2-3)
11. **StatusCard component**
12. **SystemInfo component**
13. **QuickActions component**
14. **Polling setup con TanStack Query**
15. **Dashboard page completa**

### Fase 4: Configuración (Semana 3-4)
16. **Config service**
17. **ConfigForm con react-hook-form + Zod**
18. **ConfigPreview**
19. **Save/Reset actions**
20. **Validación de errores**

### Fase 5: Runtime & Docker (Semana 4)
21. **Runtime status component**
22. **Restart button con confirmación**
23. **Uptime display**
24. **Docker container status**
25. **Docker actions**

### Fase 6: Logs (Semana 5)
26. **LogViewer component**
27. **SSE connection**
28. **Log filtering**
29. **Search functionality**
30. **Download/Clear controls**

### Fase 7: Polish (Semana 6)
31. **Animaciones y transiciones**
32. **Skeleton loaders**
33. **Empty states**
34. **Responsive design**
35. **Testing**

---

## 10. Commands de Desarrollo

```bash
# Install
npm install

# Development
npm run dev

# Build
npm run build

# Preview
npm run preview

# Lint
npm run lint

# Type check
npm run typecheck
```

---

## 11. Próximos Pasos Inmediatos

1. ☐ Crear proyecto Vite + TypeScript
2. ☐ Instalar dependencias
3. ☐ Configurar Tailwind
4. ☐ Configurar Shadcn/UI
5. ☐ Crear tipos (src/types/)
6. ☐ Crear API client
7. ☐ Crear layout base
8. ☐ Crear Dashboard con StatusCard

**Tiempo estimado total: 6 semanas**
