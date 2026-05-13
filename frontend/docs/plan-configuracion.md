# Plan: Configuración de Node-RED desde Frontend

## Objetivo
Permitir al usuario editar la configuración de Node-RED desde la UI web y persistirla en `settings.js`.

---

## 1. Estado Actual

### Backend ✅ (ya implementado)
- `GET /api/config` → Lee `config.json`
- `POST /api/config` → Guarda config + genera `settings.js`
- `POST /api/config/validate` → Valida con Zod

### Frontend ⚠️ (básico)
- Página Configuration existe pero es muy simple
- No guarda cambios
- No muestra validación

---

## 2. Tareas a Implementar

### Fase 1: Mejora del Formulario de Configuración

#### 1.1 Actualizar tipos TypeScript
```typescript
// frontend/src/types/index.ts
// Agregar tipos más completos para la config
```

#### 1.2 Crear componente ConfigForm
- Campos editables para cada propiedad
- Validación en tiempo real
- Estados: loading, editing, saving, success, error

#### 1.3 Implementar guardado
- Conectar con `POST /api/config`
- Mostrar toast de éxito/error
- Invalidar queries después de guardar

---

### Fase 2: Validación y Feedback

#### 2.1 Validación en tiempo real
- Llamar a `POST /api/config/validate` mientras el usuario escribe (debounced)
- Mostrar errores inline

#### 2.2 Preview JSON
- Mostrar JSON de la configuración actual
- Actualizar en tiempo real

#### 2.3 Historial de cambios
- Trackear cambios pendientes
- Mostrar botón de "Descartar cambios"

---

### Fase 3: Persistencia y Restart

#### 3.1 Guardar configuración
- POST a `/api/config`
- Validar respuesta
- Mostrar feedback

#### 3.2 Preview de settings.js
- Opcional: mostrar cómo quedó el `settings.js` generado

#### 3.3 Restart automático (opcional)
- Después de guardar, preguntar si quiere reiniciar Node-RED
- Llamar a `POST /api/runtime/restart`

---

## 3. Archivos a Modificar

### Frontend

| Archivo | Acción |
|---------|--------|
| `src/types/index.ts` | Completar tipos de configuración |
| `src/pages/Configuration.tsx` | Reescribir completamente |
| `src/components/config/ConfigForm.tsx` | Crear componente |
| `src/components/config/ConfigPreview.tsx` | Crear componente |
| `src/services/index.ts` | Ya existe, revisar |
| `src/lib/utils.ts` | Agregar utilitarios si es necesario |

---

## 4. Estructura de la Página Configuration

```
┌─────────────────────────────────────────────────────────────┐
│  Configuration                              [Save] [Reset]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────┐  ┌─────────────────────────────┐  │
│  │     Config Form     │  │      JSON Preview          │  │
│  │                     │  │                             │  │
│  │  Port: [____]       │  │  {                         │  │
│  │                     │  │    "uiPort": 1880,         │  │
│  │  Projects: [toggle] │  │    "loggingLevel": "info" │  │
│  │                     │  │  }                         │  │
│  │  Log Level: [▼]     │  │                             │  │
│  │                     │  │                             │  │
│  │  Flow File: [____]  │  │                             │  │
│  │                     │  │                             │  │
│  └─────────────────────┘  └─────────────────────────────┘  │
│                                                             │
│  [Errores de validación si hay]                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Tipos de Configuración Node-RED

```typescript
interface NodeRedConfig {
  // Basic Settings
  uiPort: number;           // Puerto del editor
  projectsEnabled: boolean;  // Habilitar proyectos
  
  // Logging
  loggingLevel: 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  
  // Files
  flowFile?: string;        // Archivo de flows
  userDir?: string;         // Directorio de usuario
  nodesDir?: string;        // Directorio de nodos
  
  // Editor Theme
  editorTheme?: {
    projects?: { enabled?: boolean };
    palette?: { catalogue?: string[] };
    menu?: { items?: string[] };
    userMenu?: boolean;
  };
  
  // Runtime State
  runtimeState?: {
    enabled?: boolean;
    file?: string;
  };
}
```

---

## 6. Orden de Implementación

### Día 1: Formulario Básico
- [ ] Completar tipos en frontend
- [ ] Reescribir Configuration.tsx
- [ ] Crear ConfigForm con campos básicos
- [ ] Conectar GET para cargar config

### Día 2: Guardado y Validación
- [ ] Implementar POST para guardar
- [ ] Validación con Zod
- [ ] Toast notifications
- [ ] Estados de loading

### Día 3: Mejoras
- [ ] Preview JSON
- [ ] Validación en tiempo real
- [ ] Botón reset
- [ ] Integrar con restart de runtime

---

## 7. Commands de Desarrollo

```bash
# Backend (debe estar corriendo)
cd backend && npm run dev

# Frontend (debe estar corriendo)
cd frontend && npx vite

# Probar endpoint
curl http://localhost:3000/api/config

# Guardar config
curl -X POST http://localhost:3000/api/config \
  -H "Content-Type: application/json" \
  -d '{"uiPort": 1885, "loggingLevel": "debug"}'
```

---

## 8. Ejemplo de Request/Response

### GET /api/config
```json
{
  "success": true,
  "data": {
    "uiPort": 1880,
    "projectsEnabled": false,
    "loggingLevel": "info",
    "flowFile": "flows.json"
  },
  "timestamp": "2026-03-10T12:00:00Z"
}
```

### POST /api/config
```json
// Request
{
  "uiPort": 1885,
  "loggingLevel": "debug"
}

// Response
{
  "success": true,
  "data": {
    "uiPort": 1885,
    "projectsEnabled": false,
    "loggingLevel": "debug",
    "flowFile": "flows.json"
  },
  "timestamp": "2026-03-10T12:00:00Z"
}
```

---

## 9. Próximo Paso Inmediato

Comenzar con la **Fase 1**:
1. Completar tipos en frontend
2. Reescribir Configuration.tsx con formulario funcional
3. Conectar con API

¿Comenzamos?
