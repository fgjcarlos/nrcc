# ─────────────────────────────────────────────────────────────────────────────
# Stage 1: Build React frontend
# ─────────────────────────────────────────────────────────────────────────────
FROM node:26-slim AS frontend-builder

RUN npm install -g pnpm@^11

WORKDIR /build/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts

COPY frontend/ ./
RUN pnpm build

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: Build Go binary
# Needs frontend/dist for go:embed
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS go-builder

WORKDIR /build

# Download deps first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy embedded frontend dist from stage 1
COPY --from=frontend-builder /build/frontend/dist ./frontend/dist

# Copy Go source
COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" -o nrcc .

# ─────────────────────────────────────────────────────────────────────────────
# Stage 3: Runtime image
# Uses official Node-RED image as base — Node.js + node-red pre-installed,
# maintained by the Node-RED team, optimised for all target architectures.
# ─────────────────────────────────────────────────────────────────────────────
FROM nodered/node-red:latest

LABEL org.opencontainers.image.title="nrcc"
LABEL org.opencontainers.image.description="Node-RED Control Center — all-in-one binary + Node-RED"
LABEL org.opencontainers.image.source="https://github.com/composedof2/nrcc"

USER root

# Copy nrcc binary from go-builder stage
COPY --chmod=755 --from=go-builder /build/nrcc /usr/local/bin/nrcc

USER node-red

# ── Data volumes ──────────────────────────────────────────────────────────────
# /data           → nrcc config (cc-users.json, cc-config.json)
#                   + Node-RED userDir (flows.json, settings.js, lib/)
#                   + nrcc backups (backups/*.zip)
# /data/node_modules → Node-RED installed nodes (can grow large; separate volume)
VOLUME ["/data"]

# ── Environment defaults ──────────────────────────────────────────────────────
ENV DATA_DIR=/data \
    NODE_RED_CMD=node-red \
    PORT=3001

# ── Ports ─────────────────────────────────────────────────────────────────────
# 3001 → nrcc web UI + API
# 1880 → Node-RED editor (proxied through nrcc or direct)
EXPOSE 3001 1880

# Override nodered/node-red ENTRYPOINT — nrcc manages node-red as child process
ENTRYPOINT ["/usr/local/bin/nrcc"]
