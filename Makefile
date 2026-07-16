BINARY      := nrcc
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X main.Version=$(VERSION)

# Docker
IMAGE       ?= ghcr.io/fgjcarlos/nrcc
PLATFORMS   ?= linux/amd64,linux/arm64,linux/arm/v7

.PHONY: build frontend-build dev dev-native dev-build dev-up dev-attach dev-logs dev-shell dev-down dev-reset test test-frontend e2e release clean \
        docker docker-local docker-push docker-run docker-compose-up docker-compose-down \
        dev-docker dev-docker-build dev-docker-down \
        reset reset-data \
        help

## Build single binary (requires frontend/dist to exist)
build: frontend-build
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

## Build frontend only
frontend-build:
	cd frontend && pnpm install && pnpm run build

## Development mode: run Go server + Vite dev server concurrently (no Docker)
dev-native:
	@echo "→ Starting dev mode natively (Go backend on :3001, Vite on :5173)"
	@trap 'kill 0' SIGINT; \
	  (cd frontend && pnpm run dev) & \
	  PORT=3001 go run . & \
	  wait

## Run tests
test:
	go test ./...

## Run frontend tests (Vitest) from the host
test-frontend:
	cd frontend && pnpm exec vitest run

## Run Playwright e2e tests (Playwright boots its own preview server)
e2e:
	cd frontend && pnpm exec playwright test

## Cross-compile for all platforms
release: frontend-build
	mkdir -p dist
	# Linux: static binary (compatible with Alpine/musl AND glibc containers)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64  go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-arm64 .
	# Pin GOARM=7 so the armv7 ABI matches the linux/arm/v7 Docker image and does
	# not silently regress to GOARM=6 if built with an older toolchain default.
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm GOARM=7 go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-armv7 .
	# macOS / Windows: standard dynamic build
	GOOS=darwin  GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .
	@ls -lh dist/

## Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	rm -rf frontend/dist/

# ─────────────────────────────────────────────────────────────────────────────
# Docker targets
# ─────────────────────────────────────────────────────────────────────────────

## Build Docker image for current platform (local dev/test)
## Uses docker buildx bake (see docker-bake.hcl) so the shared frontend + go
## build stages in Dockerfile.base are built once and consumed by the
## release and server-local targets.
docker:
	docker buildx bake --load release

## Build Docker image using local images (no registry pull needed)
## Requires: make release first (produces dist/nrcc-linux-amd64)
docker-local: release
	docker buildx bake --load local

## Build and push multi-arch image to registry (requires docker buildx + login)
docker-push:
	docker buildx bake \
	  --set '*.platform=linux/amd64,linux/arm64,linux/arm/v7' \
	  --push \
	  release \
	  --tag $(IMAGE):$(VERSION) \
	  --tag $(IMAGE):latest

## Run nrcc container locally (quick test)
docker-run:
	docker run --rm \
	  -p 3001:3001 -p 1880:1880 \
	  -e JWT_SECRET=dev-secret \
	  -v nrcc_data:/data \
	  -v nrcc_backups:/data/backups \
	  -v nrcc_node_modules:/data/node_modules \
	  $(IMAGE):latest

## Start with docker compose (detached)
docker-compose-up:
	docker compose up -d

## Stop docker compose stack
docker-compose-down:
	docker compose down

# ─────────────────────────────────────────────────────────────────────────────
# Dev Docker targets (hot reload — for development)
#
# All targets run on the host and shell out to `docker compose -f
# docker-compose.dev.yml`. The compose file mounts the repo into the
# containers, so changes to Go source are rebuilt by air, and changes
# to the React source are hot-reloaded by Vite. No image rebuild is
# needed for normal dev work; only run `dev-build` after changing
# `Dockerfile.dev` itself.
# ─────────────────────────────────────────────────────────────────────────────

## Build the dev Docker image (run once, or after Dockerfile.dev changes)
dev-build:
	docker compose -f docker-compose.dev.yml build

## Bring the dev stack up in detached mode and stream logs
dev-up: dev-build
	docker compose -f docker-compose.dev.yml up -d
	@echo ""
	@echo "✓ Dev stack up. Open: http://localhost:5173 (UI), http://localhost:3001 (API), http://localhost:1880 (Node-RED)"
	@echo "  Stream logs with: make dev-logs"
	@echo "  Open backend shell with: make dev-shell"
	@echo "  Stop with: make dev-down"

## Start dev environment in foreground (attach + stream logs)
dev-attach:
	docker compose -f docker-compose.dev.yml up

## Stream logs from the dev stack
dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

## Open an interactive shell in the backend container
dev-shell:
	docker compose -f docker-compose.dev.yml exec backend sh

## Stop the dev stack (keeps volumes)
dev-down:
	docker compose -f docker-compose.dev.yml down

## Tear down the dev stack AND remove its volumes (clean slate)
dev-reset:
	docker compose -f docker-compose.dev.yml down -v
	-docker volume rm nrcc_dev_data nrcc_dev_backups nrcc_dev_node_modules 2>/dev/null || true
	-docker volume rm nrcc_frontend_node_modules nrcc_go_mod_cache nrcc_go_build_cache 2>/dev/null || true

# Backwards-compat aliases — `dev` is the long-standing native target; the
# Docker stack is reached via `dev-up` / `dev-attach` / `dev-docker`.
dev: dev-native
dev-docker-build: dev-build
dev-docker: dev-attach
dev-docker-down: dev-down

# ─────────────────────────────────────────────────────────────────────────────
# Reset targets — volver al punto 0
# ─────────────────────────────────────────────────────────────────────────────

## Reset completo: detiene contenedores, borra volúmenes Docker, datos locales y artefactos de build
reset:
	@echo "⚠️  Esto borrará TODO: data/, volúmenes Docker, Node-RED, artefactos de build."
	@read -p "   ¿Continuar? [y/N] " confirm && { [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; } || { echo "Cancelado."; exit 1; }
	@echo ""
	@echo "→ Deteniendo stacks de Docker..."
	-docker compose down -v 2>/dev/null || true
	-docker compose -f docker-compose.dev.yml down -v 2>/dev/null || true
	@echo "→ Eliminando volúmenes Docker nombrados (si quedan)..."
	-docker volume rm nrcc_data nrcc_backups nrcc_node_modules 2>/dev/null || true
	-docker volume rm nrcc_dev_data nrcc_dev_backups nrcc_dev_node_modules 2>/dev/null || true
	-docker volume rm nrcc_frontend_node_modules nrcc_go_mod_cache nrcc_go_build_cache 2>/dev/null || true
	@echo "→ Eliminando datos locales (data/, tmp/)..."
	rm -rf data/ tmp/
	@echo "→ Eliminando artefactos de build..."
	rm -f $(BINARY)
	rm -rf dist/ frontend/dist/
	@echo ""
	@echo "✓ Reset completo. Proyecto en estado inicial — listo para empezar desde cero."

## Reset solo datos locales (sin tocar Docker ni build artifacts)
##   Útil cuando se corre nrcc de forma nativa (make dev) y se quiere limpiar solo la data
reset-data:
	@echo "⚠️  Esto borrará ./data/ y ./tmp/ (usuarios, flows, config, Node-RED nodes)."
	@read -p "   ¿Continuar? [y/N] " confirm && { [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; } || { echo "Cancelado."; exit 1; }
	rm -rf data/ tmp/
	@echo "✓ Datos locales eliminados."

help:
	@echo "Available targets:"
	@echo "  build              - Build nrcc binary with embedded frontend"
	@echo "  frontend-build     - Build React frontend only"
	@echo "  dev                - Start dev servers natively (alias for dev-native)"
	@echo "  dev-native         - Start dev servers natively (no Docker)"
	@echo "  test               - Run Go tests"
	@echo "  test-frontend      - Run Vitest frontend tests"
	@echo "  e2e                - Run Playwright e2e tests (boots its own preview server)"
	@echo "  release            - Cross-compile for all platforms"
	@echo "  clean              - Remove build artifacts"
	@echo ""
	@echo "  docker             - Build Docker image (local)"
	@echo "  docker-local       - Build using local pre-compiled binary"
	@echo "  docker-push        - Build + push multi-arch image to registry"
	@echo "  docker-run         - Run container locally (quick test)"
	@echo "  docker-compose-up  - Start production stack with docker compose"
	@echo "  docker-compose-down- Stop production docker compose stack"
	@echo ""
	@echo "  dev-build          - Build the dev Docker image"
	@echo "  dev-up             - Start dev stack (detached) + show URLs"
	@echo "  dev-attach         - Start dev stack (foreground, attached logs)"
	@echo "  dev-logs           - Stream dev stack logs"
	@echo "  dev-shell          - Open shell in backend container"
	@echo "  dev-down           - Stop dev stack (keep volumes)"
	@echo "  dev-reset          - Stop dev stack + remove volumes"
	@echo "  dev-native         - Start dev mode without Docker (alias for old 'dev')"
	@echo "  dev                - Alias for dev-native"
	@echo "  dev-docker-build   - Alias for dev-build"
	@echo "  dev-docker         - Alias for dev-attach"
	@echo "  dev-docker-down    - Alias for dev-down"
	@echo ""
	@echo "  reset              - ⚠️  Reset COMPLETO: contenedores + volúmenes + data/ + build"
	@echo "  reset-data         - ⚠️  Reset solo datos locales (data/ y tmp/)"
