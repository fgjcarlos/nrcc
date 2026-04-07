# Fase 10 — Logs, Diagnóstico y Soporte

## Implementation Status

✅ **Phase 10 is fully implemented** as of April 7, 2026 (Semana 4 - Tests and Documentation Pass).

### What was built

| Component | File(s) | Status |
|-----------|---------|--------|
| Log model | `internal/model/log.go` | ✅ |
| Job model | `internal/model/job.go` | ✅ |
| Doctor model | `internal/model/doctor.go` | ✅ |
| Support bundle model | `internal/model/support.go` | ✅ |
| Log ring buffer | `internal/service/log_buffer.go` | ✅ |
| LogService | `internal/service/logs.go` | ✅ |
| JobsService | `internal/service/jobs.go` | ✅ |
| JobContext | `internal/service/job_context.go` | ✅ |
| DoctorService | `internal/service/doctor.go` | ✅ |
| SupportBundleService | `internal/service/support_bundle.go` | ✅ |
| Sanitizer | `internal/security/sanitizer.go` | ✅ |
| Diagnostics API | `internal/server/diagnostics.go` | ✅ |
| CLI: nrcc doctor | `cmd/doctor.go` | ✅ |
| CLI: nrcc logs | `cmd/logs.go` | ✅ |
| CLI: nrcc support | `cmd/support.go` | ✅ |
| Frontend: DiagnosticsPage | `frontend/src/App.tsx` | ✅ |
| Frontend: types + API | `frontend/src/api.ts` | ✅ |

### Tests implemented

| Test file | Tests | Status |
|-----------|-------|--------|
| `internal/security/sanitizer_test.go` | 9 test cases | ✅ All passing |
| `internal/service/log_buffer_test.go` | 10 test cases | ✅ All passing |
| `internal/service/logs_test.go` | 7 test cases | ✅ All passing |
| `internal/service/jobs_test.go` | 19 test cases + JobContext | ✅ All passing |
| `internal/service/doctor_test.go` | 11 test cases | ✅ All passing |

### API Endpoints

- `GET /api/diagnostics/report` — Run all doctor checks
- `GET /api/diagnostics/logs` — Query structured logs
- `GET /api/diagnostics/jobs` — Query job history
- `POST /api/diagnostics/export` — Generate support bundle ZIP

---

Documento operativo para desarrollar la fase 10 definida en [IMPLEMENTATION_PLAN.md](/home/composedof2/Dev/Codex/Node-RED Control Center/IMPLEMENTATION_PLAN.md).

## Objetivo

Facilitar soporte y mantenimiento local con visibilidad suficiente para diagnosticar fallos comunes sin exponer secretos ni obligar al usuario a inspeccionar archivos manualmente.

La fase 10 debe convertir las capacidades actuales de estado y logs en un subsistema de soporte coherente:

- visor de logs recientes útil de verdad
- historial de jobs y operaciones críticas
- export de bundle de diagnóstico
- ampliación real de `nrcc doctor`
- pantalla de diagnóstico para soporte local

## Contexto Actual

El proyecto ya dispone de piezas que esta fase debe aprovechar:

- `ProcessManager` con ring buffer de logs del runtime
- `GET /api/runtime/status`
- `GET /api/runtime/logs`
- `GET /api/system/info`
- `OperationLock` para operaciones exclusivas
- auditoría básica en auth
- frontend con página de logs y polling periódico

La fase 10 no parte de cero. Su objetivo es consolidar, estructurar y exportar información ya disponible, y añadir la que falta para soporte real.

## Alcance

### Incluido

- logs recientes del runtime con timestamps y nivel
- logs de jobs y operaciones críticas
- resumen de estado del sistema y checks de diagnóstico
- export de bundle de soporte descargable
- ampliación de `nrcc doctor` con salida humana y JSON
- vista de diagnóstico en frontend

### No incluido

- streaming remoto a terceros
- observabilidad distribuida
- integración con SaaS de logs
- telemetría automática
- subida automática de bundles

## Principios

- local-first
- cero dependencia de red para diagnosticar
- sin captura accidental de secretos
- export reproducible y fácil de compartir
- toda operación crítica debe dejar rastro
- el soporte debe poder trabajar con un único bundle

## Problemas que Debe Resolver

- el usuario sabe que "algo falla" pero no qué componente está fallando
- los logs del runtime existen, pero no están normalizados ni correlacionados
- una operación como restore, update o install puede fallar sin contexto suficiente
- `doctor` valida precondiciones básicas, pero no produce un informe de soporte completo
- soporte necesita pedir muchos datos manualmente

## Entregables

### 1. Log Store Normalizado

Crear un servicio interno de logs con formato estructurado y persistencia acotada.

Tipos de eventos mínimos:

- `runtime.stdout`
- `runtime.stderr`
- `runtime.lifecycle`
- `job.started`
- `job.finished`
- `job.failed`
- `operation.locked`
- `operation.released`
- `doctor.check`
- `auth.audit`
- `config.apply`
- `backup.create`
- `backup.restore`
- `library.install`
- `library.uninstall`
- `update.check`
- `update.apply`

Campos mínimos por evento:

- `id`
- `timestamp`
- `level`
- `source`
- `event`
- `message`
- `operationId`
- `jobId`
- `metadata`

Niveles mínimos:

- `debug`
- `info`
- `warn`
- `error`

Persistencia propuesta:

- archivo `logs/app.log` en JSONL
- rotación simple por tamaño y número de archivos
- ring buffer en memoria para lectura rápida reciente

## 2. Historial de Jobs y Operaciones

Persistir un registro resumido de operaciones críticas para soporte.

Cada job debe guardar:

- `id`
- `type`
- `status`
- `startedAt`
- `finishedAt`
- `triggeredBy`
- `summary`
- `error`

Jobs mínimos a registrar:

- restart runtime
- backup
- restore
- npm install
- npm uninstall
- update check
- update apply
- config apply

Persistencia propuesta:

- SQLite en `nrcc.db`
- tabla separada de auditoría simple
- retención configurable con límite inicial conservador

## 3. Bundle de Diagnóstico

Implementar export de soporte descargable desde UI y CLI.

Nombre sugerido:

- `nrcc-support-YYYYMMDD-HHMMSS.zip`

Contenido mínimo:

- `manifest.json`
- `runtime-status.json`
- `system-info.json`
- `doctor.json`
- `operations-status.json`
- `recent-logs.jsonl`
- `jobs.json`
- `config-summary.json`
- `environment-summary.json`
- `version.txt`

Contenido opcional según disponibilidad:

- últimos logs de restore o update
- metadatos de backups
- versión instalada de Node-RED y dependencias relevantes

Reglas de sanitización:

- no incluir cookies
- no incluir hashes de sesión
- no incluir passwords
- no incluir secretos completos de `.env.managed`
- no incluir `flows_cred.json`
- redacción parcial de valores sensibles

Regla recomendada:

- exportar nombres de variables sensibles y reemplazar valor por `REDACTED`

## 4. `nrcc doctor` Ampliado

La fase 10 debe convertir `doctor` en herramienta de diagnóstico operativo, no solo de preinstalación.

Checks mínimos:

- presencia de `node`
- presencia de `npm`
- directorio de datos
- permisos de lectura y escritura
- presencia de Node-RED instalado
- integridad de archivos mínimos
- puerto configurado libre u ocupado por el proceso esperado
- salud HTTP del runtime
- existencia y legibilidad de `config.json`
- existencia y legibilidad de `.env.managed`
- espacio libre razonable en disco
- estado de base de datos SQLite
- estado de `portless` si está habilitado

Salidas requeridas:

- salida legible por humanos
- salida JSON estructurada
- código de salida no cero si hay fallos críticos

Interfaz sugerida:

```bash
nrcc doctor
nrcc doctor --json
nrcc doctor --export
```

Comportamiento sugerido de `--export`:

- ejecuta checks
- genera bundle de soporte
- imprime ruta final

## 5. Pantalla de Diagnóstico

Añadir una vista específica en frontend para soporte local.

Secciones mínimas:

- resumen de salud general
- checks de `doctor`
- logs recientes filtrables
- historial de jobs
- información del sistema
- botón de export

Capacidades mínimas:

- filtrar por nivel
- filtrar por origen
- buscar texto libre
- copiar bloque de error
- descargar bundle
- mostrar hora relativa y absoluta

## Diseño Técnico

### Backend

Servicios nuevos o ampliados:

- `internal/service/logs.go`
- `internal/service/doctor.go`
- `internal/service/support_bundle.go`
- `internal/service/jobs.go` o extensión del sistema actual de operaciones

Modelos sugeridos:

- `model.LogEntry`
- `model.JobRecord`
- `model.DoctorReport`
- `model.DoctorCheck`
- `model.SupportBundleManifest`

Handlers o rutas sugeridas:

- `GET /api/diagnostics/report`
- `GET /api/diagnostics/logs`
- `GET /api/diagnostics/jobs`
- `POST /api/diagnostics/export`

Rutas existentes a reaprovechar:

- `/api/runtime/status`
- `/api/runtime/logs`
- `/api/system/info`
- `/api/operations/status`

### Frontend

Extender la navegación actual con una página `diagnostics` o ampliar la actual `logs` hasta convertirla en una vista de soporte.

La opción recomendada es separarla:

- `logs`: foco en runtime
- `diagnostics`: foco en soporte global

## API Propuesta

### `GET /api/diagnostics/report`

Respuesta:

```json
{
  "generatedAt": "2026-04-06T10:30:00Z",
  "overallStatus": "degraded",
  "checks": [
    {
      "id": "node_binary",
      "label": "Node.js installed",
      "status": "pass",
      "severity": "critical",
      "message": "node found at /usr/bin/node"
    }
  ]
}
```

### `GET /api/diagnostics/logs?limit=200&level=error&source=runtime`

Respuesta:

```json
{
  "lines": [
    {
      "timestamp": "2026-04-06T10:31:10Z",
      "level": "error",
      "source": "runtime",
      "event": "runtime.stderr",
      "message": "EADDRINUSE: address already in use 127.0.0.1:1880"
    }
  ]
}
```

### `GET /api/diagnostics/jobs?limit=50`

Respuesta:

```json
{
  "jobs": [
    {
      "id": "job_123",
      "type": "update.apply",
      "status": "failed",
      "startedAt": "2026-04-06T10:00:00Z",
      "finishedAt": "2026-04-06T10:01:14Z",
      "summary": "Node-RED update to 4.1.0",
      "error": "runtime health check failed after update"
    }
  ]
}
```

### `POST /api/diagnostics/export`

Respuesta:

```json
{
  "fileName": "nrcc-support-20260406-103500.zip",
  "path": "/home/user/.nrcc/data/support/nrcc-support-20260406-103500.zip",
  "generatedAt": "2026-04-06T10:35:00Z"
}
```

## Persistencia

### Archivos

- `~/.nrcc/data/logs/app.log`
- `~/.nrcc/data/logs/app.log.1`
- `~/.nrcc/data/support/*.zip`

### SQLite

Tablas sugeridas:

- `job_history`
- `doctor_runs`
- `audit_events` si no existe una estructura suficiente

La estrategia recomendada es mixta:

- logs detallados en JSONL rotado
- historial resumido e indexable en SQLite

## Seguridad

- todas las rutas de diagnóstico deben requerir sesión autenticada
- el export debe registrar auditoría
- el bundle debe quedar en directorio privado del usuario
- el contenido exportado debe pasar por sanitización explícita
- los valores sensibles nunca deben ocultarse por convención implícita; debe existir lista de redacción

Sensibles por nombre:

- `password`
- `secret`
- `token`
- `key`
- `cookie`
- `session`

## Observabilidad Local

El sistema debe poder responder rápido a estas preguntas:

- ¿Node-RED está corriendo?
- ¿está sano o arrancó degradado?
- ¿qué operación crítica fue la última?
- ¿qué falló exactamente?
- ¿el problema es configuración, puerto, dependencia o permisos?

## Estrategia de Implementación

### Paso 1

Normalizar logs del runtime sin romper `ProcessManager`.

Trabajo:

- añadir timestamps reales
- añadir nivel y source
- preservar compatibilidad con la UI actual

### Paso 2

Persistir jobs y eventos críticos.

Trabajo:

- registrar operaciones sensibles
- guardar resultado y error
- exponer consulta paginada básica

### Paso 3

Construir `doctor` extendido como servicio reutilizable por CLI y API.

Trabajo:

- mover checks a estructuras comunes
- compartir salida entre CLI y backend

### Paso 4

Implementar soporte bundle.

Trabajo:

- crear manifest
- recoger artefactos
- sanitizar
- comprimir

### Paso 5

Crear pantalla de diagnóstico y export desde UI.

## Definition of Done

- existe un visor de logs con filtros útiles
- las operaciones críticas dejan registro persistente
- `nrcc doctor` detecta fallos comunes del runtime y del entorno
- el usuario puede generar un bundle de soporte desde UI y CLI
- el bundle no incluye secretos ni credenciales
- soporte puede diagnosticar un fallo común con el bundle sin pedir acceso manual al equipo

## Casos de Aceptación

### Caso 1. Puerto ocupado

- `doctor` detecta conflicto
- la UI lo muestra como fallo crítico
- el bundle incluye el check fallido y logs recientes

### Caso 2. Node-RED no instalado

- `doctor` marca fallo crítico
- no se produce falso positivo de runtime sano

### Caso 3. Restore fallido

- el job queda registrado como `failed`
- el error aparece en historial y en el bundle

### Caso 4. Variable sensible presente

- el bundle exporta el nombre de la variable
- el valor sale redactado

## Riesgos

- crecer demasiado en alcance y convertir fase 10 en reescritura de observabilidad
- exportar información sensible por una sanitización incompleta
- mezclar logs detallados con eventos de negocio sin modelo claro
- degradar rendimiento si toda lectura depende de archivos grandes

## Decisiones Recomendadas

- usar JSONL para logs de aplicación
- usar SQLite para historial resumido e indexable
- mantener el bundle en `.zip`
- separar `diagnostics` de `logs` en frontend
- implementar `doctor` como servicio reutilizable por CLI y API

## Dependencias

Esta fase depende funcionalmente de:

- fase 2 para estado y logs del runtime
- fase 3 para auth y auditoría
- fase 4 para la superficie de UI
- fases 5 a 9 para tener operaciones críticas que registrar

Puede empezarse parcialmente antes de completar todas las fases previas, pero su valor máximo aparece cuando ya existen backups, librerías y updates.
