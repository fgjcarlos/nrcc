# syntax=docker/dockerfile:1.7
# nrcc — multi-stage build. Stage `builder` builds the React frontend
# (pnpm); stage `go-builder` builds the Go binary; stage `runtime` is the
# node-red minimal base image with nrcc + entrypoint layered on top.
#
# One compose service / one container owns one nrcc + one node-red
# process — see ADR 0003.
#
# ponytail: cache mounts (BuildKit `--mount=type=cache`) keep pnpm and
# Go module caches across builds. They do NOT bloat the final image;
# they only accelerate repeated builds. Bump them to a tiny
# /tmp/pnpm-store and /tmp/go-build inside the stage.
FROM node:26-slim AS builder

RUN npm install -g pnpm@11.12.0 --no-audit --no-fund

WORKDIR /build/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --ignore-scripts
COPY frontend/ ./
RUN pnpm build

FROM golang:1.26-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY --from=builder /build/frontend/dist ./frontend/dist
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nrcc .

FROM nodered/node-red:5.0.1-minimal
LABEL org.opencontainers.image.title="nrcc" \
      org.opencontainers.image.description="Node-RED Control Center — all-in-one" \
      org.opencontainers.image.source="https://github.com/fgjcarlos/nrcc"
# Run the entrypoint as root so it can chown the runtime volume, then
# drop to `node-red` inside the entrypoint script before exec'ing nrcc.
# The nrcc binary itself does not perform privilege drop; see
# docker/entrypoint.sh for the su invocation.
USER root
COPY --chmod=755 --from=go-builder /build/nrcc /usr/local/bin/nrcc
COPY --chmod=755 docker/entrypoint.sh /usr/local/bin/nrcc-entrypoint.sh

VOLUME ["/data"]
ENV DATA_DIR=/data NODE_RED_CMD=node-red PORT=3001
EXPOSE 3001 1880

# ponytail: wget is present in the nodered/node-red:5.0.1-minimal base
# image, so we reuse it instead of pulling curl. Bump start-period if
# the runtime ever grows heavier initialization.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:3001/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/nrcc-entrypoint.sh"]
