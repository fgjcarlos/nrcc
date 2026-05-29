BINARY      := nrcc
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X main.Version=$(VERSION)

# Docker
IMAGE       ?= ghcr.io/composedof2/nrcc
PLATFORMS   ?= linux/amd64,linux/arm64,linux/arm/v7

.PHONY: build frontend-build dev test release clean \
        docker docker-local docker-push docker-run docker-compose-up docker-compose-down \
        dev-docker dev-docker-build dev-docker-down \
        reset reset-data \
        install-local \
        help

## Build single binary (requires frontend/dist to exist)
build: frontend-build
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

## Build frontend only
frontend-build:
	cd frontend && pnpm install && pnpm run build

## Development mode: run Go server + Vite dev server concurrently
dev:
	@echo "→ Starting dev mode (Go backend on :3001, Vite on :5173)"
	@trap 'kill 0' SIGINT; \
	  (cd frontend && pnpm run dev) & \
	  PORT=3001 go run . & \
	  wait

## Run tests
test:
	go test ./...

## Cross-compile for all platforms
release: frontend-build
	mkdir -p dist
	# Linux: static binary (compatible with Alpine/musl AND glibc containers)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64  go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm    go build -tags netgo -ldflags="$(LDFLAGS) -extldflags=-static" -o dist/$(BINARY)-linux-armv7 .
	# macOS / Windows: standard dynamic build
	GOOS=darwin  GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .
	@ls -lh dist/

## Build and install local binary as systemd service on this machine
install-local: build
	sudo ./$(BINARY) install

## Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	rm -rf frontend/dist/

# ─────────────────────────────────────────────────────────────────────────────
# Docker targets
# ─────────────────────────────────────────────────────────────────────────────

## Build Docker image for current platform (local dev/test)
docker:
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

## Build Docker image using local images (no registry pull needed)
## Requires: make release first (produces dist/nrcc-linux-amd64)
docker-local: release
	docker build -f Dockerfile.local -t nrcc:local .

## Build and push multi-arch image to registry (requires docker buildx + login)
docker-push:
	docker buildx build \
	  --platform $(PLATFORMS) \
	  --build-arg VERSION=$(VERSION) \
	  -t $(IMAGE):$(VERSION) \
	  -t $(IMAGE):latest \
	  --push \
	  .

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
# ─────────────────────────────────────────────────────────────────────────────

## Build dev Docker image (run once, or after Dockerfile.dev changes)
dev-docker-build:
	docker compose -f docker-compose.dev.yml build

## Start dev environment with hot reload
##   backend  → :3001  (Go + air, rebuilds on .go file changes)
##   frontend → :5173  (Vite HMR)
##   node-red → :1880  (child process of backend)
dev-docker:
	docker compose -f docker-compose.dev.yml up

## Stop dev environment
dev-docker-down:
	docker compose -f docker-compose.dev.yml down

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
	@echo "  dev                - Start dev servers natively (Go + Vite)"
	@echo "  test               - Run Go tests"
	@echo "  release            - Cross-compile for all platforms"
	@echo "  install-local      - Build and install nrcc as a local systemd service"
	@echo "  clean              - Remove build artifacts"
	@echo ""
	@echo "  docker             - Build Docker image (local)"
	@echo "  docker-local       - Build using local pre-compiled binary"
	@echo "  docker-push        - Build + push multi-arch image to registry"
	@echo "  docker-run         - Run container locally (quick test)"
	@echo "  docker-compose-up  - Start production stack with docker compose"
	@echo "  docker-compose-down- Stop production docker compose stack"
	@echo ""
	@echo "  dev-docker-build   - Build dev Docker image (once)"
	@echo "  dev-docker         - Start hot-reload dev environment in Docker"
	@echo "  dev-docker-down    - Stop dev Docker environment"
	@echo ""
	@echo "  reset              - ⚠️  Reset COMPLETO: contenedores + volúmenes + data/ + build"
	@echo "  reset-data         - ⚠️  Reset solo datos locales (data/ y tmp/)"
