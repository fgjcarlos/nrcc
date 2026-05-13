# Plan de migración de `nrcc`

## Objetivo acordado

Reorientar `nrcc` para que deje de comportarse por defecto como un wrapper que crea y controla su propia instancia de Node-RED, y pase a ser:

- una aplicación que arranca primero en modo consola,
- diagnostica el sistema anfitrión,
- detecta si Node-RED ya existe y cómo está instalado,
- puede guiar la instalación de dependencias con autorización del usuario,
- y ofrece después una UI web para administrar la instalación detectada.

La virtud principal del producto pasa a ser la gestión visual de `settings.js`, manteniendo además las funciones actuales del frontend.

## Lo que se planificó

### 1. Nuevo arranque de producto

- Sustituir el arranque actual por un bootstrap CLI.
- Ejecutar comprobaciones del entorno antes de levantar la UI.
- Detectar:
  - `node`
  - `npm`
  - `node-red`
  - Docker
  - Docker Compose
  - ruta real de `settings.js`
  - modo de instalación: nativo o Docker

### 2. Nuevo modelo operativo

- `nrcc` no debe arrancar Node-RED automáticamente por defecto.
- Debe tratar Node-RED como una instalación detectada, no como un proceso siempre “propiedad” de `nrcc`.
- Mantener un modo opcional gestionado por `nrcc` para compatibilidad.

### 3. Gestión de instalación

- Si falta `Node.js`, avisar y ofrecer instalarlo con autorización explícita del usuario.
- Si falta Node-RED, ofrecer instalación:
  - nativa
  - Docker
- Dejar la decisión final al usuario.

### 4. Configuración y `settings.js`

- Hacer de `settings.js` el centro de la propuesta de valor.
- Permitir edición visual para campos conocidos.
- Permitir edición avanzada/raw del archivo real.
- Crear backup antes de escribir cambios.
- Preservar al máximo la instalación existente.

### 5. Frontend

- Mantener las funciones actuales ya existentes.
- Mostrar el estado real del host detectado.
- Indicar ruta y origen del `settings.js` activo.
- Adaptar runtime/configuración al nuevo modelo.

### 6. Validación

- Verificar backend con tests.
- Verificar frontend con build.
- Revisar regresiones del flujo actual.

## Lo que se realizó

### Backend y arranque

- Se añadió un flujo de bootstrap/doctor en consola en `main.go`.
- Se añadieron comandos de arranque:
  - flujo por defecto con bootstrap CLI
  - `doctor`
  - `setup`
- `nrcc` ya no arranca Node-RED automáticamente por defecto.
- Se mantiene compatibilidad opcional con gestión directa del runtime mediante:
  - `NRCC_MANAGE_NODE_RED=true`

### Detección del host

Se implementó una capa de inspección del entorno que detecta:

- `node`
- `npm`
- `node-red`
- Docker
- `docker compose`
- instalación Node-RED nativa o Docker
- contenedor Node-RED cuando aplica
- ruta del `settings.js`
- estado general del host

Archivos principales:

- `internal/model/bootstrap.go`
- `internal/service/host.go`

### Nuevos endpoints

Se añadieron endpoints para exponer el nuevo estado del sistema:

- estado de bootstrap/host
- lectura del `settings.js` real
- guardado del `settings.js` real

Archivos principales:

- `internal/handler/bootstrap.go`
- `internal/handler/settings.go`
- `internal/server/server.go`

### Configuración

Se modificó la capa de configuración para:

- usar el `settings.js` detectado como archivo real de trabajo,
- generar backup antes de guardar,
- mantener sincronía básica con `config.json`,
- soportar lectura raw del archivo,
- soportar edición avanzada desde frontend.

Archivo principal:

- `internal/service/config.go`

### Runtime

Se adaptó el estado del runtime para contemplar:

- instalación detectada,
- modo de instalación,
- si está gestionado o no por `nrcc`,
- estado “detected” además de `running/stopped`.

Archivos principales:

- `internal/model/runtime.go`
- `internal/handler/runtime.go`

### Frontend

Se mantuvieron las pantallas principales existentes y además:

- se añadió consumo del estado de bootstrap,
- se muestra el entorno detectado en dashboard,
- se muestra la ruta activa de `settings.js`,
- se añadió editor raw avanzado de `settings.js` en configuración,
- se añadió soporte de nuevos tipos compartidos.

Archivos principales:

- `frontend/src/shared/types/index.ts`
- `frontend/src/services/index.ts`
- `frontend/src/pages/Dashboard.tsx`
- `frontend/src/pages/Configuration.tsx`

### Validación realizada

- `go test ./...` pasando.
- `npm run build` pasando.
- Se instaló `frontend/node_modules` con `npm install` porque faltaba `vite` en el entorno local.

## Lo que falta por hacer

### 1. Instalación asistida realmente robusta

La base existe, pero falta endurecerla:

- soportar más escenarios reales de Linux,
- manejar distintos gestores de paquetes,
- mejorar errores y rollback,
- cubrir instalación de Node.js/Node-RED con mayor fiabilidad.

### 2. Soporte Docker más completo

Falta profundizar en:

- `docker compose`
- detección de layouts reales
- volúmenes personalizados
- distintas rutas de `/data`
- reinicio/control más seguro de contenedores existentes

### 3. Edición visual avanzada de `settings.js`

El editor visual actual sigue siendo parcial respecto al objetivo final:

- no modela todas las claves útiles de `settings.js`,
- no interpreta estructuras arbitrarias complejas,
- no garantiza round-trip perfecto de configuraciones muy personalizadas.

### 4. Conservación total de instalaciones existentes

Falta mejorar la estrategia para:

- parsear `settings.js` real con más fidelidad,
- preservar bloques desconocidos de forma segura también desde el modo visual,
- evitar discrepancias entre editor visual y editor raw.

### 5. Flujo de bootstrap más completo en UI

Hoy la parte más fuerte del bootstrap está en consola. Falta:

- reflejar mejor el onboarding en la web,
- guiar al usuario desde la UI según dependencias ausentes,
- exponer acciones de instalación o resolución desde la interfaz.

### 6. Cobertura de pruebas

Falta ampliar tests para:

- detección del host,
- casos nativo vs Docker,
- backup y guardado de `settings.js`,
- sincronización entre raw y visual,
- flujos de bootstrap e instalación.

### 7. Documentación de usuario

Falta actualizar:

- `README.md`
- instrucciones de arranque
- variables de entorno nuevas
- diferencias entre modo detectado y modo gestionado
- limitaciones conocidas de la edición de `settings.js`

## Estado actual resumido

La migración está iniciada y funcional, pero no cerrada.

Estado real del proyecto ahora:

- el cambio de dirección principal ya está implementado,
- el producto ya no depende de crear su propia instancia de Node-RED por defecto,
- ya existe bootstrap CLI, detección del host y edición raw del `settings.js`,
- pero todavía falta trabajo para considerar la nueva visión completamente terminada y sólida para producción.

## Siguiente fase recomendada

Orden recomendado de continuación:

1. Robustecer instalación/detección de host.
2. Mejorar preservación real de `settings.js`.
3. Completar soporte Docker/Compose.
4. Convertir el bootstrap en una experiencia web + consola coherente.
5. Ampliar tests y documentación.
