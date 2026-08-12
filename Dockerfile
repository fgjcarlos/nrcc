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
#
# ref: node:26-slim — pinned to digest for supply-chain integrity.
# Dependabot (docker ecosystem, weekly) bumps the digest when upstream
# changes. See issue #593 and auditoria/devops-security.md §2.1.
FROM node:26-slim@sha256:4ebb5ace66f15a24c14c492e01a8beeed4fddf970a856109f5126e703e5fe503 AS builder

RUN npm install -g pnpm@11.12.0 --no-audit --no-fund

WORKDIR /build/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --ignore-scripts
COPY frontend/ ./
RUN pnpm build

# ref: golang:1.26-alpine — pinned to digest for supply-chain integrity.
# Dependabot (docker ecosystem, weekly) bumps the digest when upstream
# changes. See issue #593 and auditoria/devops-security.md §2.1.
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY --from=builder /build/frontend/dist ./frontend/dist
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nrcc .

# ref: nodered/node-red:5.0.4-24-minimal — community image, pinned to
# digest. Dependabot (docker ecosystem, weekly) bumps the digest when
# upstream changes. See issue #593 and auditoria/devops-security.md §2.1.
FROM nodered/node-red:5.0.4-24-minimal@sha256:7d57985fde220f7d223e008a2e8f97a46dcd432009fcdfaaaf00ed3e37a540ab
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

# ponytail: wget is present in the nodered/node-red:5.0.4-24-minimal
# base image (same major.minor as the runtime FROM above), so we
# reuse it instead of pulling curl. Bump start-period if the runtime
# ever grows heavier initialization.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:3001/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/nrcc-entrypoint.sh"]
