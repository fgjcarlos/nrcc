# Node-RED Control Center — Propuesta Técnica Final

Herramienta local-first para administrar una instancia Node-RED desde una interfaz web moderna, con backend en Go, frontend en React y acceso local estable mediante hostname legible. El proyecto está pensado para ejecutarse en el equipo del usuario, no como panel multi-tenant expuesto a internet.

El objetivo es construir el mejor producto posible para uso local: robusto, seguro dentro de su contexto, fácil de operar y con foco en operaciones reales sobre Node-RED. La prioridad no es maximizar features en el MVP, sino asegurar consistencia, control y mantenibilidad.

---

## Decisiones de Arquitectura

### Decisión Principal

- **Backend en Go**
- **Frontend en React + Vite + TypeScript**
- **Producto local-first**
- **`portless` como capa de acceso local por hostname**
- **Node-RED gestionado como proceso local**

### Qué significa esto

- El backend no será una API Express convencional, sino un **orquestador local**.
- Node-RED correrá como proceso hijo controlado por el backend.
- La UI se servirá desde el propio binario Go.
- El acceso preferente será por URL local estable, por ejemplo `https://control-center.localhost`, en lugar de depender de recordar puertos.
- La persistencia será local, combinando SQLite para estado interno del panel y archivos para el estado propio de Node-RED y los backups.

### Por qué esta es la mejor base

- Go encaja mejor que Express en control de procesos, archivos, locks, jobs y operaciones sensibles.
- Un binario único simplifica instalación, distribución y soporte.
- `portless` mejora mucho la UX local sin complicar el núcleo del sistema.
- El producto queda coherente con el objetivo real: herramienta de administración local, no plataforma SaaS.

---

## Stack Tecnológico

| Capa | Tecnología | Notas |
|------|-----------|-------|
| **Backend** | Go 1.22+ | Binario único, buen manejo de concurrencia y procesos |
| **Router HTTP** | `net/http` + `chi` | Ligero, claro y suficiente |
| **Frontend** | React + Vite + TypeScript | UI rica, rápida de iterar |
| **UI** | Tailwind CSS + DaisyUI | Válido para un panel local si se usa con criterio |
| **State** | TanStack React Query + Zustand | Separación clara entre server state y UI state |
| **Auth** | Cookie de sesión + bcrypt | Preferible a JWT en `localStorage` |
| **Persistencia interna** | SQLite | Usuarios, sesiones, auditoría, jobs |
| **Persistencia Node-RED** | Archivos + snapshots | Flows, settings, backups, manifests |
| **Runtime Node-RED** | `os/exec` | Proceso hijo gestionado por Go |
| **Acceso local** | `portless` | Hostnames locales estables con HTTPS local |
| **Distribución** | binario Go + wrapper npm opcional | `npx` como UX, no como núcleo |

**Requisito del usuario**: Node.js instalado localmente para ejecutar Node-RED y gestionar paquetes npm.

---

## Rol de `portless`

`portless` aporta valor real en este proyecto, pero debe ocupar el lugar correcto.

### Sí debe usarse para

- dar una URL local estable al panel
- evitar fricción con puertos cambiantes
- mejorar la experiencia de desarrollo local
- facilitar cookies seguras y pruebas con HTTPS local

### No debe usarse para

- definir la arquitectura interna del producto
- sustituir el runtime del backend
- convertirse en dependencia obligatoria del core operativo

### Decisión práctica

- El producto debe poder funcionar con puerto normal
- La experiencia recomendada será con `portless`
- La URL por defecto puede ser algo como `https://control-center.localhost`

---

## Arquitectura General

```text
$ nrcc start
      │
      ▼
┌─────────────────────────────────────────────┐
│               Proceso principal Go          │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │ HTTP server                           │  │
│  │ - SPA React embebida                  │  │
│  │ - API REST local                      │  │
│  │ - auth, jobs, config, backups         │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │ ProcessManager                        │  │
│  │ - arranca Node-RED                    │  │
│  │ - captura logs                        │  │
│  │ - restart / stop / health             │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │ JobManager                            │  │
│  │ - updates                             │  │
│  │ - installs npm                        │  │
│  │ - backups / restore                   │  │
│  │ - locks exclusivos                    │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
      │
      ├──> Node-RED (child process)
      │
      └──> ~/.nrcc/data/
           - flows.json
           - flows_cred.json
           - settings.js
           - config.json
           - nrcc.db
           - .env.managed
           - backups/
           - manifests/
           - logs/
           - package.json
           - package-lock.json
```

### Acceso local

- `portless` expone la app con hostname local estable
- El backend escucha en un puerto local interno
- El usuario entra por hostname, no por puerto

---

## Modelo Operativo

El backend es responsable de coordinar operaciones que nunca deben pisarse entre sí.

### Operaciones exclusivas

- restart de Node-RED
- actualización de Node-RED
- instalación o desinstalación de librerías npm
- restore de backup
- regeneración de `settings.js`

### Reglas

- solo una operación destructiva o mutante a la vez
- todas las operaciones críticas generan log
- toda acción sensible tiene estado visible en UI
- las escrituras de archivos deben ser atómicas
- antes de una restauración o update se hace backup preventivo

### Estados globales del sistema

- `healthy`
- `starting`
- `restarting`
- `updating`
- `installing`
- `restoring`
- `degraded`
- `locked`

---

## Estructura del Proyecto

```text
nrcc/
├── main.go
├── go.mod
├── go.sum
├── Makefile
├── embed.go
│
├── cmd/
│   ├── start.go
│   ├── stop.go
│   └── version.go
│
├── internal/
│   ├── server/
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── spa.go
│   │
│   ├── handler/
│   │   ├── auth.go
│   │   ├── config.go
│   │   ├── runtime.go
│   │   ├── logs.go
│   │   ├── libraries.go
│   │   ├── updates.go
│   │   ├── envvars.go
│   │   ├── backups.go
│   │   ├── flows.go
│   │   ├── ai.go
│   │   └── system.go
│   │
│   ├── service/
│   │   ├── auth.go
│   │   ├── config.go
│   │   ├── process.go
│   │   ├── jobs.go
│   │   ├── libraries.go
│   │   ├── updates.go
│   │   ├── envvars.go
│   │   ├── backups.go
│   │   ├── flows.go
│   │   ├── ai.go
│   │   ├── logs.go
│   │   └── system.go
│   │
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── logger.go
│   │   ├── recovery.go
│   │   └── ratelimit.go
│   │
│   ├── model/
│   │   ├── response.go
│   │   ├── user.go
│   │   ├── config.go
│   │   ├── backup.go
│   │   ├── job.go
│   │   └── flow.go
│   │
│   ├── platform/
│   │   ├── npm.go
│   │   ├── process.go
│   │   └── fs.go
│   │
│   └── security/
│       ├── crypto.go
│       ├── session.go
│       └── audit.go
│
├── frontend/
│   ├── src/
│   └── dist/
│
└── npm/
    ├── package.json
    ├── install.js
    └── bin/
```

---

## Funcionalidades

### 1. Autenticación y Usuarios

- bootstrap inicial solo si no existe ningún usuario en SQLite
- sesión con cookie `HttpOnly` + `Secure` + `SameSite=Strict`
- passwords con bcrypt
- usuarios, sesiones y auditoría básica en SQLite
- roles recomendados:
  - `admin`
  - `operator`
  - `viewer`
- rate limiting en login
- registro mínimo de auditoría para accesos y acciones sensibles

### 2. Gestión de Configuración

- no editar `settings.js` libremente
- usar `config.json` como modelo declarativo soportado
- generar `settings.js` desde plantilla y valores validados
- diff previo antes de aplicar cambios
- marcar si el cambio requiere restart

### 3. Runtime Node-RED

- arrancar, detener y reiniciar Node-RED
- capturar stdout y stderr
- uptime, PID, estado, versión
- health check interno
- cola de restart segura

### 4. Librerías npm

- listar paquetes reales con `npm ls --json --depth=0`
- instalar y desinstalar con validación estricta
- mostrar progreso y logs de salida
- impedir operaciones paralelas incompatibles
- advertir si la acción requiere restart

### 5. Variables de Entorno

- editor de variables gestionadas por el panel
- almacenamiento en archivo administrado por la herramienta
- aplicación controlada al runtime de Node-RED
- separación entre variables internas del panel y variables del runtime gestionado

### 6. Backups y Restore

- backups manuales
- backups automáticos en fase posterior
- snapshot comprimido + manifest JSON
- backup preventivo antes de restore o update
- validación de integridad antes de restaurar
- preservación explícita de `credentialSecret`

### 7. Actualizaciones

- consulta de versión instalada y versión objetivo
- política de actualización por confirmación manual
- backup previo
- actualización controlada
- rollback si Node-RED no arranca sano

### 8. Logs y Diagnóstico

- logs del proceso Node-RED
- logs de jobs del sistema
- histórico corto visible desde UI
- exportación de logs para soporte local

### 9. Flows

- lectura de `flows.json`
- métricas básicas por flow
- listado de nodos, conexiones y tipos
- vista de detalle

### 10. IA y Asistencia

- fuera del MVP crítico
- primero análisis no destructivo
- nada de escritura automática de flows en primera fase
- si se integra IA, debe ser opt-in y explícita

---

## API

Formato uniforme:

```json
{ "success": true, "data": {}, "timestamp": "ISO8601" }
```

```json
{ "success": false, "error": { "code": "ERROR_CODE", "message": "..." }, "timestamp": "ISO8601" }
```

### Endpoints principales

| Grupo | Método | Ruta | Descripción |
|-------|--------|------|-------------|
| Auth | POST | `/api/auth/register` | bootstrap primer usuario |
| Auth | POST | `/api/auth/login` | iniciar sesión |
| Auth | POST | `/api/auth/logout` | cerrar sesión |
| Auth | GET | `/api/auth/me` | sesión actual |
| Runtime | GET | `/api/runtime/status` | estado Node-RED |
| Runtime | POST | `/api/runtime/restart` | reiniciar runtime |
| Runtime | GET | `/api/runtime/logs` | logs recientes |
| Config | GET | `/api/config` | leer configuración |
| Config | POST | `/api/config/validate` | validar propuesta |
| Config | POST | `/api/config/apply` | aplicar configuración |
| Libraries | GET | `/api/libraries` | listar paquetes |
| Libraries | POST | `/api/libraries/:name` | instalar paquete |
| Libraries | DELETE | `/api/libraries/:name` | eliminar paquete |
| Env | GET | `/api/env` | listar variables |
| Env | POST | `/api/env` | guardar variables |
| Backups | GET | `/api/backups` | listar snapshots |
| Backups | POST | `/api/backups/create` | crear backup |
| Backups | POST | `/api/backups/:id/restore` | restaurar backup |
| Updates | GET | `/api/updates/status` | comprobar actualización |
| Updates | POST | `/api/updates/apply` | aplicar actualización |
| Flows | GET | `/api/flows` | listar flows |
| Flows | GET | `/api/flows/:id` | detalle flow |
| System | GET | `/api/system/info` | información local |
| Jobs | GET | `/api/jobs` | cola y estado de trabajos |

---

## Seguridad

Este proyecto es local, pero no por eso debe tratarse como trivial.

### Reglas mínimas

- nunca guardar credenciales en texto plano
- nunca usar `localStorage` para sesión principal
- rate limit en autenticación
- protección CSRF si se usan cookies de sesión
- validación estricta de nombres de paquetes npm
- todas las rutas mutantes requieren auth
- auditoría básica de acciones críticas

### Suposición de confianza

La herramienta está diseñada para uso local y de alta confianza en la máquina del usuario. No está pensada como panel multiusuario abierto a red pública.

---

## Persistencia

### Directorio base recomendado

- `~/.nrcc/data`

### Archivos clave

- `config.json`
- `settings.js`
- `nrcc.db`
- `.env.managed`
- `flows.json`
- `flows_cred.json`
- `package.json`
- `package-lock.json`
- `backups/`
- `logs/`
- `manifests/`

### Reglas de persistencia

- escritura a archivo temporal + rename atómico
- SQLite para el estado interno del panel
- manifest por operación sensible
- hash de snapshots
- backup previo a operaciones destructivas

---

## Distribución

### Opción recomendada

- binario Go por plataforma
- wrapper npm opcional para UX tipo `npx`

### Comando deseado

```bash
npx nrcc start
```

### Qué hace `start`

1. valida entorno local
2. inicializa directorio de datos
3. arranca backend
4. arranca Node-RED
5. registra o utiliza hostname local con `portless`
6. abre o imprime URL local del panel

---

## MVP Recomendado

- arranque local estable
- auth segura con bootstrap inicial
- dashboard de estado
- gestión de configuración soportada
- restart controlado
- variables de entorno gestionadas
- backups manuales y restore seguro
- gestión básica de librerías npm
- logs recientes

## Fase 2

- scheduler de backups
- updates con rollback
- cola de trabajos visible
- métricas históricas
- export de diagnósticos

## Fase 3

- análisis de flows
- recomendaciones asistidas
- IA opt-in
- detección de patrones reutilizables

---

## Conclusión

La mejor dirección para este proyecto es clara:

- **Go como backend**
- **React como frontend**
- **producto local-first**
- **Node-RED gestionado como proceso local**
- **`portless` como mejora fuerte de experiencia local**

Esta combinación produce una herramienta más coherente, más robusta y más alineada con el tipo de operaciones que realmente necesita el producto.
