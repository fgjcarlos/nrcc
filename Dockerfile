# syntax=docker/dockerfile:1.7
# ─────────────────────────────────────────────────────────────────────────────
# Release / production image.
#
# The frontend + go build stages are in Dockerfile.base; this file only
# carries the runtime image (Node-RED upstream) and the nrcc wiring.
# Build with:
#   docker buildx bake release
# ─────────────────────────────────────────────────────────────────────────────
FROM nodered/node-red:latest

LABEL org.opencontainers.image.title="nrcc"
LABEL org.opencontainers.image.description="Node-RED Control Center — all-in-one binary + Node-RED"
LABEL org.opencontainers.image.source="https://github.com/fgjcarlos/nrcc"

USER root

# Copy nrcc binary from the base build (see docker-bake.hcl).
COPY --chmod=755 --from=base /build/nrcc /usr/local/bin/nrcc

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
