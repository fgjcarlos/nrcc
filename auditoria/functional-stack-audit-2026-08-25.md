# Auditoría funcional full-stack — 2026-08-25

## Alcance

Revisión de los contratos entre React, la API Go y el proceso gestionado de
Node-RED, con foco en las acciones de páginas individuales y en la validez de
la cobertura Playwright existente.

## Conclusión ejecutiva

La causa principal no era un fallo aislado de Node-RED, sino una falsa señal de
calidad: los E2E interceptaban todas las rutas `/api/*` y devolvían fixtures.
Por ello no podían detectar rutas inexistentes, respuestas con un contrato
distinto ni cambios que nunca llegaban al proceso de Node-RED.

Se añadió una suite separada contra el stack Docker real. Su criterio de éxito
no se limita a un toast: comprueba el documento de flows vivo, el PID antes y
después de reiniciar y una variable creada en NRCC consumida por un endpoint
HTTP ejecutado dentro de Node-RED.

## Hallazgos y correcciones

| Severidad | Hallazgo | Corrección |
|---|---|---|
| Crítica | Playwright simulaba el 100 % de la API. | Nueva suite `stack.spec.ts` sin mocks, ejecutada contra NRCC y Node-RED reales. |
| Crítica | Una ruta `/api` inexistente caía al SPA y respondía HTML con 200. | Fallback API JSON 404 y defensa adicional en el interceptor Axios. |
| Crítica | `/api/flows` devolvía el array plano de Node-RED, mientras la UI esperaba resúmenes por pestaña; detalle y métricas tampoco coincidían. | Adaptación backend a `FlowList`, `FlowDetail` y `FlowMetrics`, con pruebas unitarias y contrato OpenAPI actualizado. |
| Alta | Reinicio, configuración y settings podían responder éxito antes de saber si Node-RED había recargado. | Acciones síncronas y errores parciales explícitos cuando se guarda pero falla el reinicio. |
| Alta | Revertir una versión sustituía `flows.json` con Node-RED activo y sin recargarlo. | Parada, snapshot, revert y arranque coordinados por el `ProcessManager`. |
| Alta | La subida de imágenes usaba ruta, campo multipart y respuesta incompatibles con el backend. | Contrato único `/api/files/upload`, campo `file`, URL pública limitada a imágenes válidas. |
| Alta | La aceptación multi-stack ignoraba los puertos parametrizados, fijaba un único `container_name` y usaba un endpoint de setup inexistente. | Puertos interpolables, nombres por proyecto Compose y bootstrap corregido para cada stack. |
| Media | El timeout global de 10 s y el `WriteTimeout` de 15 s cortaban instalaciones y restores válidos. | Timeouts explícitos por operación y límite HTTP superior coherente con los handlers. |
| Media | El dashboard mostraba el estado del contenedor, no el del proceso Node-RED, y abría siempre `localhost`. | Estado desde `/api/runtime/history` y URL derivada del host/protocolo actuales. |
| Media | Instalar/desinstalar una librería ocultaba el fallo de recarga. | El fallo de reinicio se propaga y la lista se refresca incluso tras un resultado parcial. |

## Cobertura añadida

- Unitarias Go para conversión del documento plano de flows y sus métricas.
- Unitaria Go para garantizar JSON 404 en rutas API desconocidas.
- Unitaria frontend para rechazar HTML 200 recibido por una llamada API.
- Playwright real: contrato de rutas desconocidas.
- Playwright real: listado y detalle de un flow desplegado mediante la API de Node-RED.
- Playwright real: el botón Reiniciar debe producir un PID distinto.
- Playwright real: una variable guardada desde la página debe ser devuelta por un flow HTTP vivo.

## Ejecución

Suite rápida con fixtures:

```bash
cd frontend
pnpm test:e2e
```

Suite full-stack (requiere el Compose levantado):

```bash
NRCC_ENCRYPTION_KEY="$(openssl rand -hex 32)" docker compose up -d --build --wait
cd frontend
pnpm test:e2e:stack
```

La workflow `Acceptance (Docker stacks)` construye la imagen, ejecuta esta
suite real y después valida aislamiento, persistencia y backup/restore entre
proyectos Compose.
