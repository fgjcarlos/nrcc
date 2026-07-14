# docker-bake.hcl — multi-target build definition for the nrcc Docker images.
#
# This file is the entry point for `docker buildx bake`. It defines the
# shared `base` target (frontend + go build stages from Dockerfile.base)
# and three consumer targets that pull the built binary from it:
#
#   - release      → Dockerfile            (nodered/node-red runtime)
#   - server-local → Dockerfile.server-local (node:26-slim + npm node-red)
#   - local        → Dockerfile.local       (host-built binary, no base)
#   - dev          → Dockerfile.dev         (dev image, no base)
#
# The `contexts.base = "target:base"` line in each consumer target
# substitutes the `base` build's filesystem for the `base` reference
# inside `COPY --from=base` — no external registry required, buildx
# passes the image in-memory between targets.
#
# Usage:
#   docker buildx bake                  # build release
#   docker buildx bake server-local
#   docker buildx bake --print          # show the resolved build plan
#
# See Dockerfile.base for the rationale on which Dockerfiles extend it
# and which are self-contained.

target "base" {
  context    = "."
  dockerfile = "Dockerfile.base"
  target     = "go-builder"
}

target "release" {
  context    = "."
  dockerfile = "Dockerfile"
  contexts   = {
    base = "target:base"
  }
  platforms  = ["linux/amd64"]
  output     = ["type=docker"]
}

target "server-local" {
  context    = "."
  dockerfile = "Dockerfile.server-local"
  contexts   = {
    base = "target:base"
  }
  platforms  = ["linux/amd64"]
  output     = ["type=docker"]
}

# `local` and `dev` do not depend on the base image: `local` copies a
# pre-built host binary, `dev` mounts source at runtime. They are
# exposed as bake targets for a uniform entry point but consume no
# shared stages.
target "local" {
  context    = "."
  dockerfile = "Dockerfile.local"
  platforms  = ["linux/amd64"]
  output     = ["type=docker"]
}

target "dev" {
  context    = "."
  dockerfile = "Dockerfile.dev"
  platforms  = ["linux/amd64"]
  output     = ["type=docker"]
}

group "default" {
  targets = ["release"]
}
