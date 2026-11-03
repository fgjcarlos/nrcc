#!/usr/bin/env python3
"""Focused tests for the nrcc release packaging scripts (issue #884).

Standard library only (unittest, hashlib, subprocess, tempfile) — the
release tooling must not introduce new dependencies.

Run with:

    python3 scripts/release/test_package.py

or:

    python3 -m unittest discover -s scripts/release -p 'test_*.py'

The suite exercises the public contract of the two scripts:

* ``scripts/release/package.sh VERSION OUTPUT_DIR`` builds exactly the
  supported Linux artifacts (``nrcc-linux-amd64``, ``nrcc-linux-arm64``,
  ``nrcc-linux-armv7``) plus a ``SHA256SUMS`` manifest and normalized
  ``*.sha256`` sidecars.
* ``scripts/release/verify.sh VERSION OUTPUT_DIR`` re-computes the
  checksums, reads the Go build metadata back out of every binary and
  performs an isolated native install/start smoke test.
"""

from __future__ import annotations

import hashlib
import os
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
RELEASE_DIR = Path(__file__).resolve().parent
PACKAGE_SH = RELEASE_DIR / "package.sh"
VERIFY_SH = RELEASE_DIR / "verify.sh"
DOCKERFILE = REPO_ROOT / "Dockerfile"

TEST_VERSION = "v0.0.0-pkgtest"

# artifact name -> (GOARCH, GOARM or None)
ARTIFACTS: dict[str, tuple[str, str | None]] = {
    "nrcc-linux-amd64": ("amd64", None),
    "nrcc-linux-arm64": ("arm64", None),
    "nrcc-linux-armv7": ("arm", "7"),
}

EXPECTED_OUTPUT_FILES = sorted(
    list(ARTIFACTS)
    + [f"{name}.sha256" for name in ARTIFACTS]
    + ["SHA256SUMS"]
)

BUILD_SETTING_RE = re.compile(r"^\s*build\s+(\S+?)=(.*)$", re.MULTILINE)


def run(cmd: list[str], **kwargs) -> subprocess.CompletedProcess:
    """Run a command and capture text output."""
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        check=False,
        **kwargs,
    )


def sha256_file(path: Path) -> str:
    """Return the hex sha256 digest of a file."""
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def parse_build_settings(metadata: str) -> dict[str, str]:
    """Parse the ``build`` settings block of ``go version -m`` output."""
    settings: dict[str, str] = {}
    for key, value in BUILD_SETTING_RE.findall(metadata):
        settings[key] = value.strip().strip('"')
    return settings


def parse_manifest(text: str) -> dict[str, str]:
    """Parse a ``sha256sum`` manifest into {filename: digest}."""
    entries: dict[str, str] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line:
            continue
        parts = line.split()
        if len(parts) != 2:
            raise AssertionError(f"malformed SHA256SUMS line: {line!r}")
        entries[parts[1]] = parts[0]
    return entries


class ReleasePackagingTest(unittest.TestCase):
    """Package/verify behaviour for the supported Linux artifacts."""

    @classmethod
    def setUpClass(cls) -> None:
        if shutil.which("go") is None:
            raise unittest.SkipTest("the go toolchain is required to package artifacts")
        if not PACKAGE_SH.is_file():
            raise AssertionError(f"missing packaging script: {PACKAGE_SH}")
        if not VERIFY_SH.is_file():
            raise AssertionError(f"missing verification script: {VERIFY_SH}")

        cls._tmp = tempfile.TemporaryDirectory(prefix="nrcc-release-test-")
        cls.out_dir = Path(cls._tmp.name) / "dist"
        result = run([str(PACKAGE_SH), TEST_VERSION, str(cls.out_dir)], cwd=REPO_ROOT)
        cls.package_result = result
        cls.package_failure = None
        if result.returncode != 0:
            # Keep the failure attached to the individual tests instead of
            # aborting the whole class, so the independent checks still run.
            cls.package_failure = (
                f"package.sh failed (exit {result.returncode}):\n"
                f"stdout:\n{result.stdout}\n"
                f"stderr:\n{result.stderr}"
            )

    @classmethod
    def tearDownClass(cls) -> None:
        tmp = getattr(cls, "_tmp", None)
        if tmp is not None:
            tmp.cleanup()

    def require_package(self) -> None:
        """Fail the current test when the packaging step did not succeed."""
        if self.package_failure is not None:
            self.fail(self.package_failure)

    # ── packaging ────────────────────────────────────────────────────────

    def test_01_package_emits_only_supported_linux_artifacts(self) -> None:
        self.require_package()
        produced = sorted(p.name for p in self.out_dir.iterdir())
        self.assertEqual(produced, EXPECTED_OUTPUT_FILES)
        for name in ARTIFACTS:
            path = self.out_dir / name
            self.assertTrue(path.is_file(), f"missing artifact {name}")
            self.assertTrue(os.access(path, os.X_OK), f"{name} is not executable")
        # Unsupported targets must not be claimed (ADR 0004 — Linux only).
        for unsupported in ("nrcc-darwin-amd64", "nrcc-darwin-arm64", "nrcc-windows-amd64.exe"):
            self.assertFalse(
                (self.out_dir / unsupported).exists(),
                f"unsupported target was produced: {unsupported}",
            )

    def test_02_checksum_manifest_and_sidecars_match_actual_bytes(self) -> None:
        self.require_package()
        manifest = parse_manifest((self.out_dir / "SHA256SUMS").read_text())
        self.assertEqual(sorted(manifest), sorted(ARTIFACTS))
        for name in ARTIFACTS:
            actual = sha256_file(self.out_dir / name)
            self.assertEqual(manifest[name], actual, f"manifest digest mismatch for {name}")
            sidecar_parts = (self.out_dir / f"{name}.sha256").read_text().split()
            self.assertEqual(len(sidecar_parts), 2, f"malformed sidecar for {name}")
            self.assertEqual(sidecar_parts[0], actual, f"sidecar digest mismatch for {name}")
            # Normalized filename convention: the sidecar refers to "nrcc".
            self.assertEqual(sidecar_parts[1], "nrcc")

    def test_03_embedded_build_metadata_reports_version_and_target(self) -> None:
        self.require_package()
        for name, (goarch, goarm) in ARTIFACTS.items():
            with self.subTest(artifact=name):
                result = run(["go", "version", "-m", str(self.out_dir / name)])
                self.assertEqual(result.returncode, 0, result.stderr)
                settings = parse_build_settings(result.stdout)
                self.assertEqual(settings.get("GOOS"), "linux")
                self.assertEqual(settings.get("GOARCH"), goarch)
                self.assertEqual(settings.get("CGO_ENABLED"), "0")
                if goarm is None:
                    self.assertNotIn("GOARM", settings)
                else:
                    self.assertEqual(settings.get("GOARM"), goarm)
                self.assertIn(
                    f"main.Version={TEST_VERSION}",
                    settings.get("-ldflags", ""),
                    f"{name} does not embed the requested version",
                )

    # ── verification ─────────────────────────────────────────────────────

    def test_04_verify_accepts_packaged_artifacts(self) -> None:
        self.require_package()
        result = run([str(VERIFY_SH), TEST_VERSION, str(self.out_dir)])
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("OK", result.stdout)

    def _copy_dist(self, tmp: str) -> Path:
        destination = Path(tmp) / "dist"
        shutil.copytree(self.out_dir, destination)
        return destination

    def test_05_verify_rejects_tampered_artifact(self) -> None:
        self.require_package()
        with tempfile.TemporaryDirectory(prefix="nrcc-release-tamper-") as tmp:
            copy = self._copy_dist(tmp)
            manifest_before = (copy / "SHA256SUMS").read_text()
            target = copy / "nrcc-linux-amd64"
            with open(target, "r+b") as handle:
                handle.seek(0)
                first = handle.read(1)
                handle.seek(0)
                handle.write(bytes([first[0] ^ 0xFF]))
            result = run([str(VERIFY_SH), TEST_VERSION, str(copy)])
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("checksum", (result.stdout + result.stderr).lower())
            # A rejected package must not be rewritten by the verifier.
            self.assertEqual((copy / "SHA256SUMS").read_text(), manifest_before)

    def test_06_verify_rejects_version_mismatch(self) -> None:
        self.require_package()
        result = run([str(VERIFY_SH), "v9.9.9-other", str(self.out_dir)])
        self.assertNotEqual(result.returncode, 0)
        combined = result.stdout + result.stderr
        self.assertIn("version", combined.lower())

    def test_verify_rejects_version_prefix_without_mutating_artifacts(self) -> None:
        self.require_package()
        with tempfile.TemporaryDirectory(prefix="nrcc-release-prefix-") as tmp:
            dist = self._copy_dist(tmp)
            before = {path.name: sha256_file(path) for path in dist.iterdir() if path.is_file()}
            accepted = run([str(VERIFY_SH), TEST_VERSION, str(dist)])
            self.assertEqual(accepted.returncode, 0, accepted.stdout + accepted.stderr)
            self.assertEqual(accepted.stderr, "")
            self.assertIn(f"all checks passed for version {TEST_VERSION}", accepted.stdout)
            result = run([str(VERIFY_SH), TEST_VERSION[:-1], str(dist)])
            after = {path.name: sha256_file(path) for path in dist.iterdir() if path.is_file()}
            self.assertEqual(after, before)
            self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
            self.assertIn(
                "[verify] FAIL: nrcc-linux-amd64: embedded version mismatch",
                result.stderr,
            )

    def test_07_verify_rejects_missing_artifact(self) -> None:
        self.require_package()
        with tempfile.TemporaryDirectory(prefix="nrcc-release-missing-") as tmp:
            copy = self._copy_dist(tmp)
            (copy / "nrcc-linux-armv7").unlink()
            result = run([str(VERIFY_SH), TEST_VERSION, str(copy)])
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("nrcc-linux-armv7", result.stdout + result.stderr)

    # ── argument handling ───────────────────────────────────────────────

    def test_08_package_rejects_invalid_arguments(self) -> None:
        result = run([str(PACKAGE_SH)])
        self.assertEqual(result.returncode, 2)
        self.assertIn("Usage", result.stderr)

        with tempfile.TemporaryDirectory(prefix="nrcc-release-args-") as tmp:
            out_dir = Path(tmp) / "dist"
            for bad_version in ("", "bad version", "v1.2.3;rm", "../escape"):
                with self.subTest(version=bad_version):
                    result = run([str(PACKAGE_SH), bad_version, str(out_dir)])
                    self.assertEqual(result.returncode, 2)
                    self.assertIn("Usage", result.stderr)
                    # Rejected input must not leave artifacts behind.
                    self.assertFalse(
                        out_dir.exists() and any(out_dir.iterdir()),
                        "rejected version left artifacts behind",
                    )

    def test_09_verify_rejects_invalid_arguments(self) -> None:
        result = run([str(VERIFY_SH)])
        self.assertEqual(result.returncode, 2)
        self.assertIn("Usage", result.stderr)

    # ── docker image versioning ─────────────────────────────────────────

    def test_10_dockerfile_declares_version_build_arg_and_matching_label(self) -> None:
        text = DOCKERFILE.read_text()
        self.assertIn("ARG VERSION=dev", text)
        self.assertIn("-X main.Version=${VERSION}", text)
        self.assertIn('org.opencontainers.image.version="${VERSION}"', text)

    def test_11_armv7_target_still_compiles(self) -> None:
        """The armv7 artifact is claimed as supported, so it must compile.

        `syscall.Sysinfo_t` reports 32-bit counters on 32-bit Linux, which
        silently breaks the linux/arm build when the host memory counters are
        assigned to 64-bit fields. This guard keeps that regression visible.
        """
        result = run(
            ["go", "build", "./..."],
            cwd=REPO_ROOT,
            env={**os.environ, "CGO_ENABLED": "0", "GOOS": "linux", "GOARCH": "arm", "GOARM": "7"},
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    @unittest.skipUnless(
        os.environ.get("NRCC_RELEASE_DOCKER_PROBE") == "1",
        "set NRCC_RELEASE_DOCKER_PROBE=1 to run the docker image version probe",
    )
    def test_12_docker_image_label_and_binary_agree_on_version(self) -> None:
        if shutil.which("docker") is None:
            self.skipTest("docker is not available")
        probe_version = "v0.0.0-dockerprobe"
        image = "nrcc-release-probe:local"
        build = run(
            [
                "docker",
                "build",
                "--build-arg",
                f"VERSION={probe_version}",
                "-t",
                image,
                ".",
            ],
            cwd=REPO_ROOT,
        )
        self.assertEqual(build.returncode, 0, build.stdout + build.stderr)

        label = run(
            [
                "docker",
                "image",
                "inspect",
                "--format",
                '{{ index .Config.Labels "org.opencontainers.image.version" }}',
                image,
            ]
        )
        self.assertEqual(label.returncode, 0, label.stderr)
        self.assertEqual(label.stdout.strip(), probe_version)

        with tempfile.TemporaryDirectory(prefix="nrcc-release-image-") as tmp:
            container = run(["docker", "create", image]).stdout.strip()
            self.assertTrue(container)
            try:
                copied = run(
                    ["docker", "cp", f"{container}:/usr/local/bin/nrcc", str(Path(tmp) / "nrcc")]
                )
                self.assertEqual(copied.returncode, 0, copied.stderr)
                metadata = run(["go", "version", "-m", str(Path(tmp) / "nrcc")]).stdout
            finally:
                run(["docker", "rm", "-f", container])
            settings = parse_build_settings(metadata)
            self.assertIn(f"main.Version={probe_version}", settings.get("-ldflags", ""))


if __name__ == "__main__":
    unittest.main(verbosity=2)
