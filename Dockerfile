# syntax=docker/dockerfile:1.7
# nrcc — single-stage build + runtime.
#   - Stage `builder`: builds the React frontend (pnpm) and the Go binary.
#   - Stage `runtime`:  FROM nodered/node-red, copies nrcc in, sets entrypoint.
#
# One compose service / one container owns one nrcc + one node-red process.
FROM node:26-slim AS builder

RUN npm install -g corepack@latest && corepack enable && corepack prepare pnpm@11.12.0 --activate

WORKDIR /build/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY frontend/ ./
RUN pnpm build

FROM golang:1.26-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY --from=builder /build/frontend/dist ./frontend/dist
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o nrcc .

FROM nodered/node-red:5.0
LABEL org.opencontainers.image.title="nrcc" \
      org.opencontainers.image.description="Node-RED Control Center — all-in-one" \
      org.opencontainers.image.source="https://github.com/fgjcarlos/nrcc"
# Run the entrypoint as root so it can chown the runtime volume, then
# drop to `node-red` inside the entrypoint script before exec'ing nrcc.
# The nrcc binary itself does not perform privilege drop; see
# docker/entrypoint.sh for the setpriv invocation.
USER root
COPY --chmod=755 --from=go-builder /build/nrcc /usr/local/bin/nrcc
COPY --chmod=755 docker/entrypoint.sh /usr/local/bin/nrcc-entrypoint.sh

VOLUME ["/data"]
ENV DATA_DIR=/data NODE_RED_CMD=node-red PORT=3001
EXPOSE 3001 1880

# ponytail: wget is present in the nodered/node-red:5.0 base image, so we
# reuse it instead of pulling curl. Bump start-period if the runtime
# ever grows heavier initialization.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:3001/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/nrcc-entrypoint.sh"]
