# Node-RED Control Center — Plan de Implementación

Plan de ejecución para construir la primera versión usable del proyecto definido en `TECHNICAL_OVERVIEW.md`.

El criterio principal es este:

- primero se construye una base local robusta
- después se añaden operaciones sensibles con control
- por último se incorporan capacidades avanzadas

---

## Objetivo del MVP

Entregar una herramienta local usable en Linux que permita:

- preparar el entorno del usuario
- arrancar y detener Node-RED
- autenticarse de forma segura
- ver estado y logs
- gestionar configuración soportada
- gestionar variables de entorno
- hacer backups y restore
- instalar y quitar librerías npm

Quedan fuera del MVP:

- IA
- automatismos de edición de flows
- soporte completo macOS
- soporte Windows
- instaladores nativos complejos

---

## Principios de Implementación

- no construir features sin base operativa
- no permitir operaciones concurrentes peligrosas
- toda mutación importante debe dejar rastro
- toda escritura de archivos debe ser atómica
- `setup` y `doctor` son parte del producto, no herramientas auxiliares

---

## Fase 0 — Bootstrap del Proyecto

### Objetivo

Crear la base del monorepo local y la toolchain de desarrollo.

### Entregables

- estructura de carpetas Go
- proyecto frontend React + Vite
- integración de build frontend -> `dist`
- `embed.FS` en Go
- `Makefile`
- formato y lint básicos
- `README` técnico mínimo

### Tareas

1. Crear módulo Go y estructura `cmd/`, `internal/`, `frontend/`
2. Crear frontend React + TypeScript
3. Añadir pipeline local:
   - `npm run build` en frontend
   - copia o consumo de `frontend/dist`
   - build del binario Go
4. Montar servidor HTTP mínimo en Go sirviendo la SPA
5. Definir respuesta JSON estándar

### Definition of Done

- `make dev` arranca backend y frontend en desarrollo
- `make build` genera binario funcional
- el binario sirve la SPA compilada

---

## Fase 1 — CLI y Setup del Entorno

### Objetivo

Resolver la instalación real del usuario local.

### Comandos iniciales

- `nrcc setup`
- `nrcc start`
- `nrcc stop`
- `nrcc doctor`
- `nrcc version`

### Comportamiento esperado

#### `nrcc setup`

- detecta sistema operativo soportado
- comprueba `node` y `npm`
- comprueba si `portless` está disponible
- explica qué falta
- pide confirmación antes de instalar o modificar nada
- crea `~/.nrcc/data`
- inicializa `package.json` local
- instala `node-red` en el directorio de datos
- genera archivos base

#### `nrcc doctor`

- valida versión de Node.js
- valida directorio de datos
- valida presencia de Node-RED local
- valida permisos de escritura
- valida estado de `portless`
- devuelve diagnóstico comprensible

#### `nrcc start`

- valida precondiciones mínimas
- arranca backend
- arranca Node-RED
- imprime URL local
- si `portless` está disponible, publica hostname recomendado

### Tareas

1. Implementar parser CLI
2. Implementar detección de dependencias
3. Implementar inicialización de `~/.nrcc/data`
4. Implementar wrapper de npm para:
   - `npm init`
   - `npm install node-red`
   - `npm ls`
5. Implementar diagnóstico estructurado
6. Definir códigos de salida CLI

### Definition of Done

- un usuario Linux limpio puede ejecutar `nrcc setup`
- el setup pide confirmación antes de instalar
- `nrcc doctor` informa claramente estado y problemas
- `nrcc start` funciona tras setup exitoso

---

## Fase 2 — Runtime Manager de Node-RED

### Objetivo

Controlar Node-RED como proceso local fiable.

### Entregables

- `ProcessManager`
- start/stop/restart
- captura de stdout/stderr
- estado del proceso
- health check

### Tareas

1. Implementar arranque de proceso con `os/exec`
2. Gestionar entorno del proceso
3. Capturar logs en ring buffer
4. Detectar salida inesperada
5. Implementar shutdown limpio con señales
6. Exponer estado por API

### Riesgos

- reinicios inestables si el proceso no libera puerto
- diferencias entre shells y plataformas
- timeouts de arranque de Node-RED

### Definition of Done

- Node-RED arranca desde el binario
- se puede reiniciar desde API y CLI
- los logs recientes son accesibles
- el backend detecta si Node-RED cae

---

## Fase 3 — Auth, Sesión y Seguridad Base

### Objetivo

Cerrar el panel con una seguridad local razonable.

### Entregables

- bootstrap inicial de usuario
- login/logout
- cookie de sesión segura
- SQLite para usuarios y estado de auth
- middleware de auth
- rate limiting
- auditoría mínima

### Tareas

1. Crear `nrcc.db` y schema inicial
2. Modelo de usuarios en SQLite
3. Hash de passwords con bcrypt
4. Cookie de sesión `HttpOnly`
5. Protección CSRF si la sesión usa cookie
6. Rate limit de login
7. Registro de eventos en SQLite:
   - login correcto
   - login fallido
   - logout
   - acciones críticas

### Definition of Done

- no existe ruta mutante sin auth
- el primer usuario solo se puede crear una vez
- la sesión sobrevive recarga de página
- login fallido repetido se limita

---

## Fase 4 — Shell del Frontend y Dashboard

### Objetivo

Tener una interfaz usable de verdad, no solo endpoints.

### Entregables

- layout principal
- login
- dashboard
- navegación lateral
- estado global de sistema
- toasts y errores

### Tareas

1. Crear layout y sistema de rutas
2. Implementar guard de sesión
3. Implementar dashboard con:
   - estado Node-RED
   - versión
   - uptime
   - estado global
4. Añadir polling o React Query
5. Añadir UX de acciones críticas

### Definition of Done

- el usuario puede iniciar sesión
- puede ver estado del sistema
- puede reiniciar Node-RED desde UI
- los errores de backend se entienden

---

## Fase 5 — Configuración Soportada

### Objetivo

Gestionar configuración sin exponer edición arbitraria de `settings.js`.

### Entregables

- `config.json` soportado por la herramienta
- generador de `settings.js`
- validación
- diff previo
- marcador de restart requerido

### Tareas

1. Definir subset de configuración soportada
2. Modelar structs Go validados
3. Crear generador de `settings.js`
4. Implementar escritura atómica
5. Implementar endpoint de validación
6. Implementar endpoint de apply
7. Añadir pantalla de configuración

### Definition of Done

- la configuración soportada se edita desde UI
- `settings.js` se genera sin intervención manual
- cambios inválidos no pisan estado previo

---

## Fase 6 — Variables de Entorno

### Objetivo

Gestionar variables del runtime de Node-RED de forma explícita y segura.

### Entregables

- archivo `.env.managed`
- CRUD desde UI
- separación entre vars internas y de runtime
- aplicación controlada en reinicio

### Tareas

1. Diseñar formato persistente
2. Validar nombres y valores
3. Implementar lectura/escritura atómica
4. Inyectar variables al proceso Node-RED
5. Mostrar impacto de cambios

### Definition of Done

- las variables gestionadas se editan desde UI
- se aplican al runtime tras restart
- no se mezclan con secretos internos del panel

---

## Fase 7 — Backups y Restore

### Objetivo

Dar seguridad operativa real al usuario.

### Entregables

- snapshots comprimidos
- manifest JSON
- backup manual
- restore con validación
- backup preventivo pre-restore

### Tareas

1. Definir formato de snapshot
2. Incluir archivos mínimos obligatorios
3. Calcular hash y metadatos
4. Implementar restore seguro
5. Verificar compatibilidad mínima
6. Añadir UI de backups

### Definition of Done

- el usuario puede crear backup desde UI
- puede restaurar un backup válido
- antes de restaurar se crea backup preventivo

---

## Fase 8 — Librerías npm

### Objetivo

Permitir gestionar extensiones Node-RED sin romper el entorno.

### Entregables

- listado real de paquetes instalados
- instalación
- desinstalación
- logs de ejecución
- exclusión mutua con otras operaciones

### Tareas

1. Implementar `npm ls --json --depth=0`
2. Validar nombre de paquete
3. Implementar `npm install`
4. Implementar `npm uninstall`
5. Integrar con `JobManager`
6. Añadir UI y estado de job

### Definition of Done

- la instalación deja traza
- no se ejecutan dos installs al mismo tiempo
- un restore o update bloquea installs

---

## Fase 9 — Actualizaciones de Node-RED

### Objetivo

Actualizar con el menor riesgo posible.

### Entregables

- detección de versión instalada
- detección de versión objetivo
- confirmación manual
- backup previo
- rollback

### Tareas

1. Leer versión instalada
2. Consultar versión disponible
3. Aplicar política de actualización
4. Crear backup previo
5. Ejecutar update
6. Verificar arranque sano
7. Revertir si falla

### Definition of Done

- una actualización exitosa deja estado consistente
- una actualización fallida recupera un estado usable

---

## Fase 10 — Logs, Diagnóstico y Soporte

### Objetivo

Facilitar soporte y mantenimiento local.

### Entregables

- visor de logs recientes
- log de jobs
- export de diagnóstico
- `nrcc doctor` ampliado

### Tareas

1. Consolidar logs de sistema
2. Añadir timestamps y niveles
3. Crear export de bundle de soporte
4. Añadir pantalla de diagnóstico

### Definition of Done

- el usuario puede exportar información útil para soporte
- el sistema permite diagnosticar fallos comunes

---

## Fase 11 — Flows

### Objetivo

Aportar visibilidad, no automatismo destructivo.

### Entregables

- lectura de flows
- métricas
- detalle por flow
- análisis determinista básico

### Definition of Done

- la UI muestra flows reales y métricas útiles

---

## Fase 12 — Roadmap Posterior

### macOS

- adaptar `setup`
- validar integración con `portless`
- revisar señales y rutas

### Windows

- replantear gestión de procesos
- revisar rutas, permisos y quoting
- validar estrategia equivalente a `portless`

### IA

- solo opt-in
- análisis no destructivo
- sin escritura automática en MVP ampliado

---

## Dependencias de Usuario por Fase

### MVP Linux

- binario `nrcc`
- Node.js y npm
- navegador moderno
- `portless` opcional

### Fase posterior

- automatización guiada de instalación de Node.js previa confirmación del usuario
- soporte oficial macOS
- soporte oficial Windows

---

## Riesgos Principales

- complejidad real del setup local
- diferencias de entorno entre usuarios Linux
- estabilidad del arranque de Node-RED
- manejo seguro de restores y updates
- dependencia parcial en herramientas externas como npm y `portless`

---

## Orden Recomendado de Ejecución

1. Fase 0
2. Fase 1
3. Fase 2
4. Fase 3
5. Fase 4
6. Fase 5
7. Fase 6
8. Fase 7
9. Fase 8
10. Fase 10
11. Fase 9
12. Fase 11
13. Fase 12

El ajuste respecto al orden natural es intencional: primero soporte, estabilidad y observabilidad; después updates; al final análisis avanzado.

---

## Criterio de Éxito del MVP

El MVP está bien si un usuario Linux puede hacer esto sin tocar terminal adicional ni editar archivos manualmente:

1. ejecutar `nrcc setup`
2. ejecutar `nrcc start`
3. abrir el panel
4. crear el primer usuario
5. revisar estado y logs
6. cambiar configuración soportada
7. cambiar variables de entorno
8. crear un backup
9. restaurarlo
10. instalar una librería npm

Si ese flujo no es sólido, el MVP aún no está terminado.
