#!/usr/bin/env python3
"""Release publication policy for nrcc (#884).

This module is the single source of truth for the decisions the release
workflow has to make before it publishes anything:

* **Version validation** — release tags and manual versions must be
  strict semver (`MAJOR.MINOR.PATCH[-prerelease][+build]`). Malformed
  tags such as `v1.2`, `v1.2.3.4` or `v01.2.3` are rejected instead of
  being published under a broken version.
* **Image channels** — a stable release publishes the exact version, the
  floating `major.minor` channel and `latest`. A prerelease publishes
  *only* its exact version, so it can never take over `latest` or the
  `major.minor` channel.
* **GitHub release flags** — prereleases are created with
  `prerelease: true` and `make_latest: false`; stable releases keep
  `prerelease: false` and `make_latest: true`.
* **Optional DockerHub target** — the DockerHub image target and the
  DockerHub registry login are driven by the *same* availability flag,
  so the workflow can never publish to a registry it did not
  authenticate against (or the other way round).

The workflow consumes this module through its command line interface,
which writes GitHub Actions output (`key=value` plus heredoc blocks) to
standard output:

    python3 scripts/release/metadata.py --tag v1.2.3 --image ghcr.io/acme/nrcc

Standard library only: the release workflow runs it on a bare runner.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass, field
from typing import Iterable, Sequence

__all__ = [
    "InvalidVersion",
    "Version",
    "parse_version",
    "channel_tags",
    "image_tags",
    "release_flags",
    "dockerhub_available",
    "image_labels",
    "policy_for",
    "render_env_output",
    "main",
]

# Strict semver, semver.org §2: numeric identifiers carry no leading
# zeroes and pre-release / build identifiers are dot separated.
_CORE_RE = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
_NUMERIC_RE = re.compile(r"^(0|[1-9][0-9]*)$")
_IDENTIFIER_RE = re.compile(r"^[0-9A-Za-z-]+$")

# Heredoc delimiter used for multi-line GitHub Actions outputs.
OUTPUT_DELIMITER = "NRCC_EOF"

IMAGE_TITLE = "nrcc"
IMAGE_DESCRIPTION = "Node-RED Control Center — all-in-one"
IMAGE_SOURCE = "https://github.com/fgjcarlos/nrcc"


class InvalidVersion(ValueError):
    """Raised when a version or release tag is not valid strict semver."""


@dataclass(frozen=True)
class Version:
    """A parsed, validated semantic version."""

    major: int
    minor: int
    patch: int
    prerelease: tuple[str, ...] = field(default_factory=tuple)
    build: tuple[str, ...] = field(default_factory=tuple)

    @property
    def core(self) -> str:
        return f"{self.major}.{self.minor}.{self.patch}"

    @property
    def is_prerelease(self) -> bool:
        return bool(self.prerelease)

    @property
    def semver(self) -> str:
        """`MAJOR.MINOR.PATCH[-prerelease]` (build metadata excluded).

        Build metadata does not take part in version precedence and is
        not valid inside a container tag, so it never reaches a tag.
        """
        text = self.core
        if self.prerelease:
            text += "-" + ".".join(self.prerelease)
        return text

    @property
    def tag(self) -> str:
        """The canonical release tag, always prefixed with `v`."""
        suffix = "+" + ".".join(self.build) if self.build else ""
        return "v" + self.semver + suffix


def _parse_identifiers(text: str, kind: str) -> tuple[str, ...]:
    parts = text.split(".")
    for part in parts:
        if not part:
            raise InvalidVersion(f"empty {kind} identifier in {text!r}")
        if not _IDENTIFIER_RE.match(part):
            raise InvalidVersion(f"invalid {kind} identifier {part!r}")
        if kind == "prerelease" and part.isdigit() and not _NUMERIC_RE.match(part):
            raise InvalidVersion(
                f"numeric {kind} identifier {part!r} must not have leading zeroes"
            )
    return tuple(parts)


def parse_version(text: str, require_v: bool = False) -> Version:
    """Parse `text` as a strict semver version.

    Args:
        text: the candidate version, optionally prefixed with `v`.
        require_v: when true the `v` prefix is mandatory (release tags).

    Raises:
        InvalidVersion: when `text` is not a valid strict semver version.
    """
    if not isinstance(text, str):
        raise InvalidVersion("version must be a string")
    if text == "":
        raise InvalidVersion("version is empty")
    if text != text.strip():
        raise InvalidVersion(f"version {text!r} has surrounding whitespace")

    has_prefix = text.startswith("v")
    if require_v and not has_prefix:
        raise InvalidVersion(f"release tag {text!r} must start with 'v'")

    body = text[1:] if has_prefix else text

    build: tuple[str, ...] = ()
    if "+" in body:
        body, _, build_text = body.partition("+")
        if "+" in build_text:
            raise InvalidVersion(f"invalid build metadata {build_text!r}")
        build = _parse_identifiers(build_text, "build")

    prerelease: tuple[str, ...] = ()
    if "-" in body:
        core_text, _, prerelease_text = body.partition("-")
        if not prerelease_text:
            raise InvalidVersion(f"empty prerelease identifier in {text!r}")
        prerelease = _parse_identifiers(prerelease_text, "prerelease")
    else:
        core_text = body

    match = _CORE_RE.match(core_text)
    if not match:
        raise InvalidVersion(
            f"{text!r} is not a strict semantic version (expected MAJOR.MINOR.PATCH)"
        )

    major, minor, patch = (int(part) for part in match.groups())
    return Version(major=major, minor=minor, patch=patch, prerelease=prerelease, build=build)


def channel_tags(version: Version) -> list[str]:
    """Return the image channels a release may publish.

    Stable releases own `latest` and the floating `major.minor` channel.
    Prereleases own their exact version only — they must never move a
    floating channel.
    """
    if version.is_prerelease:
        return [version.semver]
    return [version.semver, f"{version.major}.{version.minor}", "latest"]


def image_tags(version: Version, targets: Iterable[str]) -> list[str]:
    """Return the fully qualified image tags for `targets`."""
    channels = channel_tags(version)
    return [f"{target}:{channel}" for target in targets for channel in channels]


def release_flags(version: Version) -> dict[str, str]:
    """Return the GitHub release flags for `version`."""
    if version.is_prerelease:
        return {"prerelease": "true", "make_latest": "false"}
    return {"prerelease": "false", "make_latest": "true"}


def dockerhub_available(username: str | None, token: str | None) -> bool:
    """Return whether the optional DockerHub credentials are usable."""
    return bool(username and username.strip()) and bool(token and token.strip())


def image_labels(version: Version, revision: str | None = None) -> dict[str, str]:
    """Return the OCI labels attached to the published image."""
    labels = {
        "org.opencontainers.image.title": IMAGE_TITLE,
        "org.opencontainers.image.description": IMAGE_DESCRIPTION,
        "org.opencontainers.image.source": IMAGE_SOURCE,
        "org.opencontainers.image.version": version.tag,
    }
    if revision:
        labels["org.opencontainers.image.revision"] = revision
    return labels


def policy_for(
    version_text: str,
    *,
    tag: bool = False,
    targets: Sequence[str] = (),
    dockerhub_targets: Sequence[str] = (),
    dockerhub_available: bool = False,
    revision: str | None = None,
) -> dict[str, object]:
    """Build the complete release policy for `version_text`.

    Args:
        version_text: the release tag (tag mode) or a manual version.
        tag: true for a tag push (publishes), false for a manual
            dispatch (always a dry run).
        targets: always published image targets, e.g. the GHCR image.
        dockerhub_targets: optional image targets, published only when
            `dockerhub_available` is true.
        dockerhub_available: whether the optional DockerHub credentials
            are present.
        revision: the commit SHA, recorded as the OCI revision label.
    """
    version = parse_version(version_text, require_v=tag)
    flags = release_flags(version)

    published_targets = list(targets)
    if dockerhub_available:
        published_targets.extend(dockerhub_targets)

    return {
        "version": version.tag,
        "semver": version.semver,
        "prerelease": flags["prerelease"],
        "make_latest": flags["make_latest"],
        # Only a tag push publishes; a manual dispatch is always dry.
        "publish": "true" if tag else "false",
        "dockerhub_available": "true" if dockerhub_available else "false",
        "channels": channel_tags(version),
        "tags": image_tags(version, published_targets),
        "labels": image_labels(version, revision=revision),
    }


def _render_heredoc(key: str, values: Sequence[str]) -> list[str]:
    if not values:
        return []
    return [f"{key}<<{OUTPUT_DELIMITER}", *values, OUTPUT_DELIMITER]


def render_env_output(policy: dict[str, object]) -> str:
    """Render the policy as GitHub Actions output lines."""
    lines = [
        f"version={policy['version']}",
        f"semver={policy['semver']}",
        f"prerelease={policy['prerelease']}",
        f"make_latest={policy['make_latest']}",
        f"publish={policy['publish']}",
        f"dockerhub_available={policy['dockerhub_available']}",
        "channels=" + ",".join(policy["channels"]),  # type: ignore[arg-type]
    ]
    lines.extend(_render_heredoc("tags", policy["tags"]))  # type: ignore[arg-type]
    labels = [f"{key}={value}" for key, value in policy["labels"].items()]  # type: ignore[union-attr]
    lines.extend(_render_heredoc("labels", labels))
    return "\n".join(lines) + "\n"


def _summarise(policy: dict[str, object], stream) -> None:
    mode = "tag push (publish)" if policy["publish"] == "true" else "manual dispatch (dry run)"
    print(f"release version : {policy['version']} (semver {policy['semver']})", file=stream)
    print(f"mode            : {mode}", file=stream)
    print(f"channels        : {','.join(policy['channels'])}", file=stream)  # type: ignore[arg-type]
    print(f"image tags      : {','.join(policy['tags'])}", file=stream)  # type: ignore[arg-type]
    print(f"dockerhub target: {policy['dockerhub_available']}", file=stream)


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="metadata.py",
        description="Compute the nrcc release publication policy.",
    )
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument(
        "--tag",
        help="release tag of a tag push; must be vMAJOR.MINOR.PATCH[-prerelease]",
    )
    source.add_argument(
        "--version",
        help="manual version of a workflow_dispatch run; always a dry run",
    )
    parser.add_argument(
        "--image",
        action="append",
        default=[],
        metavar="TARGET",
        help="image target that is always published (repeatable)",
    )
    parser.add_argument(
        "--dockerhub-image",
        action="append",
        default=[],
        metavar="TARGET",
        help="optional image target, published only when the credentials exist (repeatable)",
    )
    parser.add_argument(
        "--dockerhub-available",
        choices=("true", "false"),
        default="false",
        help="whether the optional DockerHub credentials are present",
    )
    parser.add_argument("--revision", default=None, help="commit SHA recorded as OCI label")
    parser.add_argument(
        "--format",
        choices=("env", "json"),
        default="env",
        help="'env' writes GitHub Actions output lines, 'json' a JSON document",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    """Entry point: compute the policy and print it."""
    args = _build_parser().parse_args(argv)

    is_tag = args.tag is not None
    raw_version = args.tag if is_tag else args.version
    # A manual version comes from a text input, so surrounding blanks are
    # forgiven; a git tag is validated exactly as it is written.
    version_text = raw_version.strip() if not is_tag else raw_version

    try:
        policy = policy_for(
            version_text,
            tag=is_tag,
            targets=args.image,
            dockerhub_targets=args.dockerhub_image,
            dockerhub_available=args.dockerhub_available == "true",
            revision=args.revision,
        )
    except InvalidVersion as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2

    if args.dockerhub_image and policy["dockerhub_available"] != "true":
        print(
            "warning: DockerHub credentials are missing, the DockerHub target is not published",
            file=sys.stderr,
        )

    if args.format == "json":
        print(json.dumps(policy, indent=2, sort_keys=True))
    else:
        sys.stdout.write(render_env_output(policy))

    _summarise(policy, sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
