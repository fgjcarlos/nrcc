# nr-cm — Node-RED Control Manager (CLI)

Herramienta CLI para gestionar una instancia Node-RED local sin Docker. Un solo comando (`npx nr-cm start`) levanta Node-RED como proceso hijo y un panel web de administración. Backend en Go, frontend React embebido en el binario.

---

## Stack Tecnológico

| Capa | Tecnología | Notas |
|------|-----------|-------|
| **Backend / CLI** | Go 1.22+ | Binario estático, sin runtime |
| **HTTP router** | `net/http` + `chi` | Stdlib-compatible, middleware nativo |
| **Frontend** | React + Vite + TypeScript | Embebido en binario via `embed.FS` |
| **UI** | Tailwind CSS + DaisyUI | Reutilizado del proyecto Docker |
| **Auth** | JWT + bcrypt | `golang-jwt/jwt` + `golang.org/x/crypto/bcrypt` |
| **Validación** | struct tags + `go-playground/validator` | Reemplaza Zod |
| **Proceso Node-RED** | `os/exec` (child process) | Node-RED corre como subprocess |
| **Persistencia** | JSON files en `~/.nr-cm/data/` | Sin base de datos |
| **Distribución** | npm (wrapper) + binario Go precompilado | `npx nr-cm` descarga binario |

**Requisito del usuario**: Node.js ≥ 18 instalado (Node-RED lo necesita para correr).

---

## Arquitectura

```
$ nr-cm start
      │
      ▼
┌─────────────────────────────────────────┐
│            Binario Go (~15 MB)          │
│                                         │
│  ┌──────────────────────────────────┐   │
│  │     HTTP Server (chi router)     │   │
│  │     :3000                        │   │
│  │                                  │   │
│  │  /           → React SPA        │   │  embed.FS
│  │  /api/*      → REST handlers    │   │
│  └──────────────────────────────────┘   │
│                                         │
│  ┌──────────────────────────────────┐   │
│  │     ProcessManager               │   │
│  │     (os/exec.Cmd)               │   │
│  │                                  │   │
│  │  Spawn: node-red -u ~/.nr-cm/   │   │
│  │         -p 1880                  │   │
│  │  Captura: stdout/stderr → logs   │   │
│  │  Señales: SIGTERM graceful stop  │   │
│  └──────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
      │                          │
      ▼                          ▼
┌────────────┐          ┌─────────────────┐
│ Node-RED   │          │ ~/.nr-cm/data/  │
│ :1880      │          │  flows.json     │
│ (child     │◀────────▶│  settings.js    │
│  process)  │          │  config.json    │
│            │          │  cc-users.json  │
│            │          │  .env           │
│            │          │  backups/       │
│            │          │  node_modules/  │
└────────────┘          └─────────────────┘
```

Un solo binario, un solo puerto expuesto (3000). Node-RED corre como child process en el puerto 1880. Los datos viven en `~/.nr-cm/data/` (configurable con `--data`).

---

## Estructura del Proyecto Go

```
nr-cm/
├── main.go                     # Entry point, CLI flags, startup
├── go.mod
├── go.sum
├── Makefile                    # Build targets por OS/arch
│
├── cmd/
│   └── root.go                 # CLI commands (start, stop, version)
│
├── internal/
│   ├── server/
│   │   ├── server.go           # HTTP server setup, middleware chain
│   │   └── routes.go           # Registro de rutas
│   │
│   ├── handler/                # HTTP handlers (equivale a controllers/)
│   │   ├── auth.go
│   │   ├── config.go
│   │   ├── runtime.go
│   │   ├── updates.go
│   │   ├── libraries.go
│   │   ├── flows.go
│   │   ├── envvars.go
│   │   ├── backups.go
│   │   ├── ai.go
│   │   ├── patterns.go
│   │   ├── logs.go
│   │   └── system.go
│   │
│   ├── service/                # Lógica de negocio
│   │   ├── auth.go             # JWT + bcrypt
│   │   ├── config.go           # R/W config.json + settings generator
│   │   ├── process.go          # ProcessManager (reemplaza DockerService)
│   │   ├── updates.go          # npm view node-red + npm update
│   │   ├── libraries.go        # npm install/uninstall en dataDir
│   │   ├── flows.go            # Lee flows.json, métricas
│   │   ├── envvars.go          # CRUD .env + sync settings.js
│   │   ├── backups.go          # Scheduler, manifests, restore
│   │   ├── ai.go               # OpenRouter/OpenAI/Anthropic/Gemini
│   │   ├── patterns.go         # Detección de patterns en flows
│   │   ├── logs.go             # Ring buffer stdout/stderr del child
│   │   └── system.go           # OS info (runtime pkg)
│   │
│   ├── middleware/
│   │   ├── auth.go             # JWT verification middleware
│   │   ├── logger.go           # Request logging (slog)
│   │   └── ratelimit.go        # Token bucket rate limiter
│   │
│   ├── model/                  # Structs + validation tags
│   │   ├── user.go
│   │   ├── config.go
│   │   ├── flow.go
│   │   ├── backup.go
│   │   └── response.go         # ApiResponse[T] genérico
│   │
│   └── platform/
│       ├── npm.go              # Wrapper: npm install, npm view, npm ls
│       └── process.go          # Child process lifecycle
│
├── frontend/                   # Submodule o copia del frontend React
│   └── dist/                   # Build pre-compilado
│
├── embed.go                    # //go:embed frontend/dist/*
│
├── dist/                       # Binarios compilados (gitignored)
│   ├── nr-cm-linux-amd64
│   ├── nr-cm-linux-arm64
│   └── nr-cm-darwin-arm64
│
└── npm/                        # Paquete npm wrapper
    ├── package.json            # bin: nr-cm → postinstall descarga binario
    ├── install.js              # Descarga binario correcto para OS/arch
    └── bin/
        └── nr-cm               # Shell script que ejecuta el binario Go
```

---

## Componentes Clave

### 1. CLI Entry Point

```go
// main.go
func main() {
    app := &cli.App{
        Name:  "nr-cm",
        Usage: "Node-RED Control Manager",
        Commands: []*cli.Command{
            {
                Name: "start",
                Flags: []cli.Flag{
                    &cli.IntFlag{Name: "port", Value: 3000, Usage: "Puerto del panel"},
                    &cli.StringFlag{Name: "data", Value: "~/.nr-cm/data", Usage: "Directorio de datos"},
                    &cli.IntFlag{Name: "nr-port", Value: 1880, Usage: "Puerto Node-RED"},
                },
                Action: startAction,
            },
            {Name: "stop", Action: stopAction},
            {Name: "version", Action: versionAction},
        },
    }
    app.Run(os.Args)
}
```

`start` hace:
1. Expandir `~/.nr-cm/data` y crear si no existe
2. Verificar que Node.js está instalado (`node --version`)
3. Si primera vez → `npm install node-red` en dataDir
4. Arrancar Node-RED como child process
5. Arrancar servidor HTTP (API + frontend estático)
6. Imprimir URLs y esperar señales

### 2. ProcessManager (reemplaza DockerService)

```go
// internal/service/process.go
type ProcessManager struct {
    cmd      *exec.Cmd
    dataDir  string
    port     int
    logs     *RingBuffer   // últimas N líneas de stdout+stderr
    mu       sync.RWMutex
    running  bool
}

func (pm *ProcessManager) Start() error {
    pm.cmd = exec.Command("npx", "node-red", "-u", pm.dataDir, "-p", strconv.Itoa(pm.port))
    pm.cmd.Env = pm.buildEnv()

    stdout, _ := pm.cmd.StdoutPipe()
    stderr, _ := pm.cmd.StderrPipe()

    if err := pm.cmd.Start(); err != nil {
        return fmt.Errorf("failed to start node-red: %w", err)
    }
    pm.running = true

    go pm.captureOutput(stdout)
    go pm.captureOutput(stderr)
    go pm.waitForExit()

    return nil
}

func (pm *ProcessManager) Stop() error    { return pm.cmd.Process.Signal(syscall.SIGTERM) }
func (pm *ProcessManager) Restart() error { pm.Stop(); time.Sleep(2*time.Second); return pm.Start() }
func (pm *ProcessManager) IsRunning() bool
func (pm *ProcessManager) GetLogs(n int) []string
func (pm *ProcessManager) Pid() int
func (pm *ProcessManager) Uptime() time.Duration
```

**Señales**: captura SIGINT/SIGTERM del proceso padre → graceful shutdown de Node-RED antes de salir.

### 3. Frontend Embebido

```go
// embed.go
package main

import "embed"

//go:embed frontend/dist/*
var frontendFS embed.FS

// En server.go:
func setupRoutes(r chi.Router) {
    // API routes primero
    r.Route("/api", func(r chi.Router) {
        // ... handlers
    })

    // Frontend SPA fallback
    fsys, _ := fs.Sub(frontendFS, "frontend/dist")
    fileServer := http.FileServer(http.FS(fsys))
    r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
        // Si el archivo existe, servirlo; sino, index.html (SPA routing)
        path := r.URL.Path
        if _, err := fs.Stat(fsys, strings.TrimPrefix(path, "/")); err != nil {
            r.URL.Path = "/"
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

El build de React se embebe en el binario Go en compile time. No hay archivos estáticos externos. El binario es autocontenido.

### 4. UpdateService (npm en vez de Docker Hub)

```go
// internal/service/updates.go
type UpdateService struct {
    dataDir string
    npm     *platform.NPM
}

func (s *UpdateService) CheckUpdate() (*UpdateStatus, error) {
    current, err := s.getCurrentVersion()   // lee node_modules/node-red/package.json
    if err != nil { return nil, err }

    latest, err := s.npm.ViewVersion("node-red")  // npm view node-red version
    if err != nil { return nil, err }

    return &UpdateStatus{
        Current:     current,
        Latest:      latest,
        UpdateReady: semver.Compare(current, latest) < 0,
    }, nil
}

func (s *UpdateService) ApplyUpdate(pm *ProcessManager) error {
    pm.Stop()
    if err := s.npm.Update("node-red", s.dataDir); err != nil {
        pm.Start() // rollback: reiniciar con versión anterior
        return err
    }
    return pm.Start()
}
```

### 5. LibraryService (npm directo)

```go
func (s *LibraryService) Install(name string) error {
    // Validar nombre (prevenir inyección de comandos)
    if !isValidPackageName(name) {
        return ErrInvalidPackageName
    }
    return s.npm.Install(name, s.dataDir)
}

func (s *LibraryService) Uninstall(name string) error {
    return s.npm.Uninstall(name, s.dataDir)
}

func (s *LibraryService) List() ([]Package, error) {
    return s.npm.ListInstalled(s.dataDir) // npm ls --json
}
```

### 6. Wrapper npm para Distribución

```go
// npm/install.js — postinstall script
const { platform, arch } = process;
const version = require('./package.json').version;

const BINARY_MAP = {
    'linux-x64':    'nr-cm-linux-amd64',
    'linux-arm64':  'nr-cm-linux-arm64',
    'darwin-arm64': 'nr-cm-darwin-arm64',
    'darwin-x64':   'nr-cm-darwin-amd64',
};

const binary = BINARY_MAP[`${platform}-${arch}`];
const url = `https://github.com/user/nr-cm/releases/download/v${version}/${binary}`;
// Descarga binario → node_modules/.bin/nr-cm-bin
```

```json
// npm/package.json
{
  "name": "nr-cm",
  "version": "0.1.0",
  "bin": { "nr-cm": "./bin/nr-cm" },
  "scripts": { "postinstall": "node install.js" },
  "os": ["linux", "darwin"],
  "cpu": ["x64", "arm64"]
}
```

El paquete npm es un thin wrapper (~5 KB). `postinstall` descarga el binario Go precompilado para la plataforma. `npx nr-cm` ejecuta el binario directamente.

---

## Equivalencias Express.js → Go

| Express.js (proyecto Docker) | Go (CLI) | Paquete/stdlib |
|------------------------------|----------|----------------|
| `express` | `net/http` + `chi` | `github.com/go-chi/chi/v5` |
| `dockerode` | `os/exec` (ProcessManager) | stdlib |
| `jsonwebtoken` | `golang-jwt/jwt/v5` | `github.com/golang-jwt/jwt/v5` |
| `bcrypt` / `bcryptjs` | `golang.org/x/crypto/bcrypt` | stdlib extension |
| `helmet` | `chi/middleware` + headers custom | chi built-in |
| `cors` | `chi/cors` | `github.com/go-chi/cors` |
| `zod` | struct tags + validator | `github.com/go-playground/validator/v10` |
| `express-rate-limit` | `golang.org/x/time/rate` | stdlib extension |
| `pino` | `log/slog` | stdlib (Go 1.21+) |
| `compression` | `chi/middleware.Compress` | chi built-in |
| `multer` | `r.FormFile()` | stdlib |
| `dotenv` | `github.com/joho/godotenv` | — |
| `express-validator` | validator + custom middleware | — |
| `serve` (frontend) | `embed.FS` + `http.FileServer` | stdlib |

### Dependencias Go totales

```
require (
    github.com/go-chi/chi/v5          // Router HTTP
    github.com/go-chi/cors             // CORS middleware
    github.com/golang-jwt/jwt/v5       // JWT
    golang.org/x/crypto                // bcrypt
    golang.org/x/time                  // rate limiter
    github.com/go-playground/validator/v10  // Validación structs
    github.com/joho/godotenv           // Leer .env
)
```

**7 dependencias directas** vs ~20 en Express. Sin transitive dependency hell.

---

## Modelo de Respuesta API (igual que versión Docker)

```go
// internal/model/response.go
type ApiResponse[T any] struct {
    Success   bool      `json:"success"`
    Data      T         `json:"data,omitempty"`
    Error     *ApiError `json:"error,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

type ApiError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func OK[T any](w http.ResponseWriter, data T) {
    json.NewEncoder(w).Encode(ApiResponse[T]{
        Success: true, Data: data, Timestamp: time.Now(),
    })
}

func Fail(w http.ResponseWriter, status int, code, msg string) {
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ApiResponse[any]{
        Success: false,
        Error:   &ApiError{Code: code, Message: msg},
        Timestamp: time.Now(),
    })
}
```

Los endpoints son idénticos al proyecto Docker. El frontend no necesita cambios en las llamadas API.

---

## Endpoints (igual que versión Docker, sin Docker-specific)

Se **eliminan**:
- `GET /api/docker/status`
- `POST /api/docker/restart`
- `GET /api/docker/info`

Se **reemplazan** internamente (misma ruta, distinta implementación):
- `GET /api/runtime/status` → lee estado del child process en vez de container inspect
- `POST /api/runtime/restart` → `ProcessManager.Restart()` en vez de Docker restart
- `GET /api/runtime/logs` → ring buffer del stdout/stderr en vez de `docker logs`
- `GET /api/updates/status` → `npm view node-red` en vez de Docker Hub API
- `POST /api/updates/apply` → `npm update node-red` en vez de pull + recreate
- `POST /api/libraries/:name` → `npm install` directo en vez de `docker exec`

Se **mantienen idénticos** (misma ruta, misma implementación):
- Todos los de Auth, Config, Flows, AI, Patterns, EnvVars, Backups, System

---

## Funcionalidades

### Mantenidas (misma lógica que versión Docker)

1. **Autenticación y Usuarios** — JWT + bcrypt, `cc-users.json`
2. **Configuración Node-RED** — Editor visual, `config.json` → `settings.js`
3. **Variables de Entorno** — CRUD `.env`, sync con `settings.js`
4. **Backups** — Manuales/automáticos, manifests, restore
5. **Flows y Análisis IA** — Métricas, detalle, análisis multi-proveedor
6. **Detección de Patrones** — AI-assisted pattern detection
7. **Dark Mode** — DaisyUI themes
8. **Sistema** — Info OS via Go `runtime` (más preciso que Node.js `os`)

### Adaptadas (distinta implementación, misma funcionalidad)

9. **Runtime** — `ProcessManager` en vez de Docker container management
10. **Actualizaciones** — `npm view/update` en vez de Docker Hub pull
11. **Librerías npm** — `npm install/uninstall` directo en dataDir
12. **Logs** — Ring buffer en memoria del stdout/stderr del child process

### Eliminadas

13. ~~Docker management~~ — No aplica (no hay Docker)

### Nuevas (específicas del CLI)

14. **First-run setup** — Auto-instala Node-RED si no existe en dataDir
15. **Graceful shutdown** — Captura señales OS, para Node-RED limpiamente
16. **Self-contained binary** — Frontend embebido, un solo archivo ejecutable

---

## Distribución

### Opción A: npm wrapper (recomendado para el UX `npx nr-cm`)

```bash
# El usuario ejecuta:
npx nr-cm start

# Lo que pasa internamente:
# 1. npm descarga paquete nr-cm de npm registry
# 2. postinstall.js detecta linux-x64
# 3. Descarga nr-cm-linux-amd64 desde GitHub Releases
# 4. Ejecuta el binario Go con los argumentos
```

**Ventaja**: familiar para usuarios Node-RED (ya tienen npm).
**Tamaño**: ~5 KB wrapper + ~15 MB binario descargado.

### Opción B: Binario directo (GitHub Releases)

```bash
# Descargar
curl -L https://github.com/user/nr-cm/releases/latest/download/nr-cm-linux-amd64 -o nr-cm
chmod +x nr-cm
./nr-cm start
```

### Opción C: Go install

```bash
go install github.com/user/nr-cm@latest
nr-cm start
```

Solo viable si el usuario tiene Go instalado (poco probable para el público objetivo).

### Build Matrix

```makefile
# Makefile
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

build-all:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -ldflags="-s -w" -o dist/nr-cm-$${platform%/*}-$${platform#*/} .; \
	done
```

| Target | OS | Arch | Tamaño estimado |
|--------|-----|------|-----------------|
| `nr-cm-linux-amd64` | Linux | x86_64 | ~12–15 MB |
| `nr-cm-linux-arm64` | Linux | ARM64 (RPi 4+) | ~12–15 MB |
| `nr-cm-darwin-amd64` | macOS | Intel | ~12–15 MB |
| `nr-cm-darwin-arm64` | macOS | Apple Silicon | ~12–15 MB |

Con `upx` compresión: ~5–7 MB por binario.

---

## Comparativa: Versión Docker vs CLI Go

| Aspecto | Docker (Express.js) | CLI (Go) |
|---------|---------------------|----------|
| **Requisitos** | Docker + Compose | Node.js ≥ 18 |
| **Instalación** | `docker compose up` | `npx nr-cm start` |
| **Procesos** | 3 contenedores | 1 binario + 1 child process |
| **RAM total** | ~600–800 MB | ~100–200 MB |
| **Disco** | ~850 MB (imágenes) | ~15 MB (binario) + ~50 MB (node-red) |
| **Puertos** | 3 (8081, 3000, 1881) | 2 (3000, 1880) |
| **Startup** | ~10–30s (containers) | ~2–3s |
| **Actualizar Node-RED** | Pull image + recreate | `npm update` + restart proceso |
| **Red interna** | Docker bridge network | localhost |
| **Aislamiento** | Contenedores aislados | Procesos del usuario |
| **Frontend** | Contenedor separado | Embebido en binario |

---

## Filesystem

```
~/.nr-cm/
├── data/                       # --data flag (configurable)
│   ├── node_modules/           # Node-RED + librerías npm instaladas
│   ├── package.json            # Dependencias npm del usuario
│   ├── flows.json              # Flows Node-RED
│   ├── flows_cred.json         # Credenciales cifradas
│   ├── settings.js             # Generado por ConfigService
│   ├── config.json             # Configuración del panel
│   ├── cc-users.json           # Usuarios del panel
│   ├── .env                    # Variables de entorno
│   └── backups/
│       ├── auto/
│       ├── manual/
│       └── pre-restore/
└── nr-cm.pid                   # PID file para `nr-cm stop`
```

---

## Dependencias Go

```
github.com/go-chi/chi/v5              v5.0.12    # Router HTTP
github.com/go-chi/cors                v1.2.1     # CORS
github.com/golang-jwt/jwt/v5          v5.2.1     # JWT tokens
golang.org/x/crypto                   v0.22.0    # bcrypt
golang.org/x/time                     v0.5.0     # Rate limiter
github.com/go-playground/validator/v10 v10.19.0  # Validación
github.com/joho/godotenv              v1.5.1     # .env parser
```

Sin dependencias transitivas pesadas. `go mod tidy` resulta en ~15 módulos totales.

---

## Seguridad

| Aspecto | Implementación |
|---------|---------------|
| JWT signing | HMAC-SHA256, secret desde config o auto-generado |
| Passwords | bcrypt cost 10 |
| Rate limiting | Token bucket (`x/time/rate`), configurable por endpoint |
| CORS | Origen restrictivo (localhost por defecto) |
| Headers | CSP, X-Frame-Options, X-Content-Type-Options via middleware |
| Command injection | Validación estricta de nombres de paquetes npm antes de `exec.Command` |
| PID file | Lock exclusivo para evitar múltiples instancias |
| Señales | SIGTERM → graceful shutdown Node-RED → cleanup PID file |
| Binario estático | Sin shell en distribución, sin dependencias dinámicas |

---

## Frontend — Cambios vs Versión Docker

| Cambio | Detalle |
|--------|---------|
| **Eliminar** | Páginas/componentes de Docker management |
| **Eliminar** | `DockerService` del API client |
| **Adaptar** | Runtime page: mostrar info de proceso en vez de container |
| **Adaptar** | Updates page: texto "actualizar paquete npm" en vez de "pull imagen" |
| **Mantener** | Todo lo demás (auth, config, flows, backups, envvars, libraries, AI, dark mode) |

Estimación: ~5% de cambios en frontend. La API es compatible — mismas rutas, mismas respuestas.

---

## Orden de Implementación

1. **Scaffold Go** — `go mod init`, estructura de directorios, Makefile
2. **CLI** — `start`, `stop`, `version` con flags (`--port`, `--data`)
3. **ProcessManager** — Spawn Node-RED, captura logs, señales, restart
4. **HTTP server + embed** — Chi router, servir frontend estático embebido
5. **Auth** — JWT + bcrypt, handlers login/register/verify, middleware
6. **Config** — R/W `config.json`, settings generator, validación
7. **Runtime** — Status del child process, restart, logs (ring buffer)
8. **Updates** — `npm view node-red` + `npm update` + restart
9. **Libraries** — Install/uninstall/list via npm CLI wrapper
10. **EnvVars** — CRUD `.env`, sync con `settings.js`
11. **Backups** — Scheduler (goroutine + ticker), manifests, restore
12. **Flows** — Leer `flows.json`, métricas, detalle
13. **AI + Patterns** — HTTP client a OpenRouter/OpenAI/etc.
14. **Frontend adapt** — Eliminar Docker pages, adaptar Runtime/Updates
15. **npm wrapper** — `install.js` + GitHub Releases + CI/CD
16. **First-run** — Auto-instalar Node-RED, setup wizard

---

## Build & Release

```bash
# Desarrollo
go run . start --port 3000

# Build frontend
cd frontend && pnpm build && cd ..

# Build binario (linux amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/nr-cm-linux-amd64 .

# Build todos los targets
make build-all

# Release: tag + GitHub Actions sube binarios a Releases
git tag v0.1.0 && git push --tags
```

### CI/CD (GitHub Actions)

```yaml
on:
  push:
    tags: ['v*']

jobs:
  build:
    strategy:
      matrix:
        include:
          - {goos: linux, goarch: amd64}
          - {goos: linux, goarch: arm64}
          - {goos: darwin, goarch: amd64}
          - {goos: darwin, goarch: arm64}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: actions/setup-node@v4
      - run: cd frontend && npm ci && npm run build
      - run: CGO_ENABLED=0 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -ldflags="-s -w" -o nr-cm-${{ matrix.goos }}-${{ matrix.goarch }}
      - uses: softprops/action-gh-release@v2
        with:
          files: nr-cm-*

  publish-npm:
    needs: build
    steps:
      - run: cd npm && npm publish
```

Cada tag `v*` compila binarios para 4 plataformas, los sube a GitHub Releases, y publica el wrapper npm que apunta a esa versión.
