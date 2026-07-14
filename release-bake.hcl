# release-bake.hcl — overrides for the GitHub release workflow.
# Merged with docker-bake.hcl by `docker buildx bake` to apply release
# tags, multi-arch platforms, and GHA cache configuration.
target "release" {
  platforms = ["linux/amd64", "linux/arm64", "linux/arm/v7"]
  output = [
    "type=image,push=true",
  ]
}
