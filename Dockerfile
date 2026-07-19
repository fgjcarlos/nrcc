# syntax=docker/dockerfile:1.7
# nrcc — single-stage build + runtime.
#   - Stage `builder`: builds the React frontend (pnpm) and the Go binary.
#   - Stage `runtime`:  FROM nodered/node-red, copies nrcc in, sets entrypoint.
#
# One compose service / one container owns one nrcc + one node-red process.
FROM node:26-slim AS builder

RUN npm install -g corepack@latest && corepack enable && corepack prepare pnpm@11.12.0 --activate

WORKDIR /build/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
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
USER root
COPY --chmod=755 --from=go-builder /build/nrcc /usr/local/bin/nrcc
USER node-red

VOLUME ["/data"]
ENV DATA_DIR=/data NODE_RED_CMD=node-red PORT=3001
EXPOSE 3001 1880

ENTRYPOINT ["/usr/local/bin/nrcc"]
