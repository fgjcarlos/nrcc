#!/usr/bin/env python3
"""Regression tests for the release publication policy (#884).

Two layers are covered:

* Policy matrix — `scripts/release/metadata.py` is the single source of
  truth for what a release version means: strict semver validation,
  image channels (`latest` / `major.minor` only for stable builds),
  GitHub release flags (`prerelease` / `make_latest`) and the optional
  DockerHub target. Those rules are asserted here directly.
* Workflow structure — `.github/workflows/release.yml`, `pr.yml` and
  `security.yml` are inspected textually (same approach as the Go
  workflow tests in `tools/langscan/workflow`) so the wiring the policy
  depends on cannot silently drift.

Standard library only: the release workflow runs this file on a bare
GitHub runner, so the suite must not need extra packages.
"""

from __future__ import annotations

import contextlib
import io
import re
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
RELEASE_DIR = Path(__file__).resolve().parent
WORKFLOW_DIR = REPO_ROOT / ".github" / "workflows"

# Keep the working tree clean: importing the policy module must not
# leave a __pycache__ directory behind.
sys.dont_write_bytecode = True

sys.path.insert(0, str(RELEASE_DIR))

import metadata  # noqa: E402  (path bootstrap must run first)


# ── helpers ──────────────────────────────────────────────────────────────────


def read_workflow(name: str) -> str:
    return (WORKFLOW_DIR / name).read_text(encoding="utf-8")


def jobs_section(text: str) -> list[str]:
    """Return the lines that belong to the top-level `jobs:` mapping."""
    lines = text.splitlines()
    for index, line in enumerate(lines):
        if line == "jobs:":
            break
    else:  # pragma: no cover - every workflow under test has jobs
        raise AssertionError("workflow has no top-level 'jobs:' section")

    body: list[str] = []
    for line in lines[index + 1 :]:
        # The section ends at the next top-level key (no indentation).
        if line.strip() and not line.startswith("  "):
            break
        body.append(line)
    return body


def job_blocks(text: str) -> dict[str, str]:
    """Map job id -> job block text for the `jobs:` section."""
    blocks: dict[str, str] = {}
    current: str | None = None
    buffer: list[str] = []

    for line in jobs_section(text):
        match = re.match(r"^  ([A-Za-z0-9_-]+):\s*$", line)
        if match:
            if current is not None:
                blocks[current] = "\n".join(buffer)
            current = match.group(1)
            buffer = [line]
        elif current is not None:
            buffer.append(line)

    if current is not None:
        blocks[current] = "\n".join(buffer)
    return blocks


def job_block(text: str, job_id: str) -> str:
    blocks = job_blocks(text)
    if job_id not in blocks:
        raise AssertionError(
            f"job {job_id!r} not found; available jobs: {sorted(blocks)}"
        )
    return blocks[job_id]


def job_needs(job_text: str) -> list[str]:
    """Return the `needs:` list of a job block."""
    lines = job_text.splitlines()
    for index, line in enumerate(lines):
        inline = re.match(r"^    needs:\s*\[(.*)\]\s*$", line)
        if inline:
            return [item.strip() for item in inline.group(1).split(",") if item.strip()]
        if re.match(r"^    needs:\s*$", line):
            needs: list[str] = []
            for candidate in lines[index + 1 :]:
                item = re.match(r"^      - (\S+)\s*$", candidate)
                if item:
                    needs.append(item.group(1))
                elif not candidate.strip():
                    continue
                else:
                    break
            return needs
        scalar = re.match(r"^    needs:\s*(\S+)\s*$", line)
        if scalar:
            return [scalar.group(1)]
    return []


def step_blocks(job_text: str) -> list[str]:
    """Split a job block into its individual step blocks."""
    steps: list[str] = []
    current: list[str] | None = None

    for line in job_text.splitlines():
        if re.match(r"^      - ", line):
            if current is not None:
                steps.append("\n".join(current))
            current = [line]
        elif current is not None:
            if not line.strip() or line.startswith("        "):
                current.append(line)
            else:
                steps.append("\n".join(current))
                current = None

    if current is not None:
        steps.append("\n".join(current))
    return steps


def step_name(step_text: str) -> str:
    match = re.search(r"^      - name: (.*)$", step_text, re.MULTILINE)
    return match.group(1).strip() if match else ""


def step_condition(step_text: str) -> str:
    match = re.search(r"^        if: (.*)$", step_text, re.MULTILINE)
    return match.group(1).strip() if match else ""


def step_run(step_text: str) -> str:
    """Return the shell body of a step (inline command or `run: |` block)."""
    lines = step_text.splitlines()

    for line in lines:
        inline = re.match(r"^\s*run: (?!\||>)(.+)$", line)
        if inline:
            return inline.group(1).strip()

    body: list[str] = []
    capturing = False

    for line in lines:
        if re.match(r"^\s*run: \|\s*$", line):
            capturing = True
            continue
        if not capturing:
            continue
        if not line.strip():
            body.append("")
            continue
        indent = len(line) - len(line.lstrip())
        if indent >= 10:
            body.append(line[10:])
            continue
        break

    return "\n".join(body)


def steps_using(job_text: str, needle: str) -> list[str]:
    return [step for step in step_blocks(job_text) if needle in step]


def all_run_scripts(text: str) -> list[str]:
    scripts: list[str] = []
    for job in job_blocks(text).values():
        for step in step_blocks(job):
            script = step_run(step)
            if script:
                scripts.append(script)
    return scripts


# ── policy matrix ────────────────────────────────────────────────────────────


class TestStrictSemverValidation(unittest.TestCase):
    VALID_TAGS = [
        "v1.2.3",
        "v0.0.1",
        "v10.20.30",
        "v1.2.3-rc.1",
        "v1.2.3-alpha.1",
        "v1.2.3-0.3.7",
        "v1.2.3-x.7.z.92",
        "v1.2.3+build.5",
        "v1.2.3-rc.1+build.5",
    ]

    MALFORMED_TAGS = [
        "v1.2",
        "v1.2.3.4",
        "v01.2.3",
        "v1.02.3",
        "v1.2.03",
        "v1.2.3-",
        "v1.2.3-rc..1",
        "v1.2.3-01",
        "v1.2.3+",
        "v1.2.3+a+b",
        "v1.2.3 ",
        " v1.2.3",
        "v1.2.3\n",
        "v",
        "vx.y.z",
        "release-1.2.3",
        "V1.2.3",
        "1.2.3",
        "",
    ]

    def test_valid_tags_are_accepted(self):
        for tag in self.VALID_TAGS:
            with self.subTest(tag=tag):
                version = metadata.parse_version(tag, require_v=True)
                self.assertTrue(version.tag.startswith("v"))

    def test_malformed_tags_are_rejected(self):
        for tag in self.MALFORMED_TAGS:
            with self.subTest(tag=tag):
                with self.assertRaises(metadata.InvalidVersion):
                    metadata.parse_version(tag, require_v=True)

    def test_tag_mode_requires_the_v_prefix(self):
        with self.assertRaises(metadata.InvalidVersion):
            metadata.parse_version("1.2.3", require_v=True)

    def test_version_mode_accepts_a_missing_v_prefix(self):
        version = metadata.parse_version("1.2.3", require_v=False)
        self.assertEqual(version.tag, "v1.2.3")
        self.assertEqual(version.semver, "1.2.3")


class TestChannelPolicy(unittest.TestCase):
    def test_stable_version_gets_exact_minor_and_latest_channels(self):
        version = metadata.parse_version("v1.2.3")
        self.assertEqual(metadata.channel_tags(version), ["1.2.3", "1.2", "latest"])

    def test_prerelease_version_gets_no_floating_channels(self):
        version = metadata.parse_version("v1.2.3-rc.1")
        self.assertEqual(metadata.channel_tags(version), ["1.2.3-rc.1"])

    def test_prerelease_never_receives_latest_or_minor_channel(self):
        for tag in ("v0.1.0-alpha", "v2.0.0-rc.1", "v1.2.3-0.3.7"):
            with self.subTest(tag=tag):
                version = metadata.parse_version(tag)
                channels = metadata.channel_tags(version)
                self.assertNotIn("latest", channels)
                self.assertFalse(
                    any(re.fullmatch(r"\d+\.\d+", channel) for channel in channels),
                    f"floating major.minor channel in {channels}",
                )

    def test_build_metadata_is_stripped_from_image_tags(self):
        version = metadata.parse_version("v1.2.3-rc.1+build.5")
        tags = metadata.image_tags(version, ["ghcr.io/acme/nrcc"])
        self.assertEqual(tags, ["ghcr.io/acme/nrcc:1.2.3-rc.1"])

    def test_image_tags_cover_every_target(self):
        version = metadata.parse_version("v1.2.3")
        tags = metadata.image_tags(version, ["ghcr.io/acme/nrcc", "docker.io/acme/nrcc"])
        self.assertEqual(
            tags,
            [
                "ghcr.io/acme/nrcc:1.2.3",
                "ghcr.io/acme/nrcc:1.2",
                "ghcr.io/acme/nrcc:latest",
                "docker.io/acme/nrcc:1.2.3",
                "docker.io/acme/nrcc:1.2",
                "docker.io/acme/nrcc:latest",
            ],
        )


class TestReleaseFlags(unittest.TestCase):
    def test_stable_release_is_published_as_latest(self):
        flags = metadata.release_flags(metadata.parse_version("v1.2.3"))
        self.assertEqual(flags["prerelease"], "false")
        self.assertEqual(flags["make_latest"], "true")

    def test_prerelease_is_marked_and_never_latest(self):
        flags = metadata.release_flags(metadata.parse_version("v1.2.3-rc.1"))
        self.assertEqual(flags["prerelease"], "true")
        self.assertEqual(flags["make_latest"], "false")


class TestDockerHubAvailability(unittest.TestCase):
    def test_both_credentials_are_required(self):
        self.assertTrue(metadata.dockerhub_available("user", "token"))
        self.assertFalse(metadata.dockerhub_available("user", ""))
        self.assertFalse(metadata.dockerhub_available("", "token"))
        self.assertFalse(metadata.dockerhub_available("", ""))
        self.assertFalse(metadata.dockerhub_available(None, "token"))

    def test_dockerhub_target_follows_the_availability_flag(self):
        version = metadata.parse_version("v1.2.3")
        without = metadata.policy_for(
            "v1.2.3",
            tag=True,
            targets=["ghcr.io/acme/nrcc"],
            dockerhub_targets=["docker.io/acme/nrcc"],
            dockerhub_available=False,
        )
        with_hub = metadata.policy_for(
            "v1.2.3",
            tag=True,
            targets=["ghcr.io/acme/nrcc"],
            dockerhub_targets=["docker.io/acme/nrcc"],
            dockerhub_available=True,
        )
        self.assertFalse(any(t.startswith("docker.io/") for t in without["tags"]))
        self.assertTrue(any(t.startswith("docker.io/") for t in with_hub["tags"]))
        self.assertTrue(version.core)


class TestPublishMode(unittest.TestCase):
    def test_build_metadata_is_preserved_in_artifact_version(self):
        policy = metadata.policy_for("v1.2.3-rc.1+build.5", tag=True)
        self.assertEqual(policy["version"], "v1.2.3-rc.1+build.5")
        self.assertEqual(policy["labels"]["org.opencontainers.image.version"], policy["version"])

    def test_tag_push_publishes(self):
        policy = metadata.policy_for("v1.2.3", tag=True)
        self.assertEqual(policy["publish"], "true")
        self.assertEqual(policy["version"], "v1.2.3")

    def test_manual_dispatch_is_always_a_dry_run(self):
        policy = metadata.policy_for("1.2.3", tag=False)
        self.assertEqual(policy["publish"], "false")
        self.assertEqual(policy["version"], "v1.2.3")


class TestCliOutput(unittest.TestCase):
    def run_cli(self, argv: list[str]) -> tuple[int, str, str]:
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = metadata.main(argv)
        return code, out.getvalue(), err.getvalue()

    def test_env_output_is_github_output_compatible(self):
        code, out, _ = self.run_cli(
            ["--version", "v1.2.3-rc.1", "--image", "ghcr.io/acme/nrcc"]
        )
        self.assertEqual(code, 0)
        self.assertIn("version=v1.2.3-rc.1\n", out)
        self.assertIn("semver=1.2.3-rc.1\n", out)
        self.assertIn("prerelease=true\n", out)
        self.assertIn("make_latest=false\n", out)
        self.assertIn("publish=false\n", out)
        self.assertIn("tags<<NRCC_EOF\n", out)
        self.assertIn("ghcr.io/acme/nrcc:1.2.3-rc.1\n", out)
        self.assertIn("labels<<NRCC_EOF\n", out)
        self.assertIn("org.opencontainers.image.version=v1.2.3-rc.1\n", out)
        # A prerelease owns no floating channel: neither `latest` nor the
        # `major.minor` tag may show up.
        self.assertNotIn(":latest", out)
        self.assertNotIn(":1.2\n", out)
        self.assertIn("channels=1.2.3-rc.1\n", out)

    def test_malformed_version_fails_with_a_clear_error(self):
        code, out, err = self.run_cli(["--tag", "v1.2"])
        self.assertEqual(code, 2)
        self.assertEqual(out, "")
        self.assertIn("error", err)

    def test_dockerhub_target_is_dropped_when_credentials_are_missing(self):
        code, out, err = self.run_cli(
            [
                "--tag",
                "v1.2.3",
                "--image",
                "ghcr.io/acme/nrcc",
                "--dockerhub-image",
                "docker.io/acme/nrcc",
                "--dockerhub-available",
                "false",
            ]
        )
        self.assertEqual(code, 0)
        self.assertNotIn("docker.io/acme/nrcc", out)
        self.assertIn("DockerHub", err)


# ── workflow structure ───────────────────────────────────────────────────────


class TestReleaseWorkflowTriggers(unittest.TestCase):
    def setUp(self):
        self.release = read_workflow("release.yml")

    def test_tag_push_and_manual_dispatch_are_the_entry_points(self):
        self.assertIn('tags:\n      - "v*"', self.release)
        self.assertIn("workflow_dispatch:", self.release)

    def test_manual_dispatch_exposes_only_a_version_input(self):
        dispatch = self.release.split("workflow_dispatch:", 1)[1]
        dispatch = dispatch.split("\njobs:", 1)[0]
        self.assertIn("version:", dispatch)
        self.assertIn("required: true", dispatch)
        self.assertIn("type: string", dispatch)
        for forbidden in ("publish:", "dry_run:", "dry-run:", "type: boolean"):
            self.assertNotIn(forbidden, dispatch)

    def test_manual_dispatch_has_no_publish_switch(self):
        # The dispatch path must not be able to publish: there is no
        # boolean input that could turn a manual run into a release.
        self.assertNotIn("github.event.inputs.publish", self.release)
        self.assertNotIn("github.event.inputs.dry_run", self.release)


class TestReleaseWorkflowGates(unittest.TestCase):
    def setUp(self):
        self.release = read_workflow("release.yml")
        self.jobs = job_blocks(self.release)

    def test_pr_and_security_workflows_are_reused(self):
        self.assertIn("uses: ./.github/workflows/pr.yml", self.release)
        self.assertIn("uses: ./.github/workflows/security.yml", self.release)

    def test_reused_workflows_declare_workflow_call(self):
        for name in ("pr.yml", "security.yml"):
            with self.subTest(workflow=name):
                self.assertIn("workflow_call:", read_workflow(name))

    def test_packaging_waits_for_both_check_workflows(self):
        needs = job_needs(self.jobs["package"])
        self.assertIn("pr-checks", needs)
        self.assertIn("security-checks", needs)

    def test_publication_waits_for_both_check_workflows(self):
        needs = job_needs(self.jobs["image"])
        self.assertIn("pr-checks", needs)
        self.assertIn("security-checks", needs)

    def test_caller_grants_security_events_permission(self):
        security_job = self.jobs["security-checks"]
        self.assertIn("security-events: write", security_job)

    def test_no_commit_status_api_gate(self):
        # The gate is the reusable workflows themselves, never a lookup of
        # check names or commit statuses.
        for forbidden in ("gh api", "check-runs", "/statuses", "commit status"):
            self.assertNotIn(forbidden, self.release.lower())

    def test_no_untrusted_event_interpolation_in_run_scripts(self):
        for script in all_run_scripts(self.release):
            self.assertNotIn("${{", script)


class TestReleaseWorkflowDryRun(unittest.TestCase):
    def setUp(self):
        self.release = read_workflow("release.yml")
        self.jobs = job_blocks(self.release)

    def publish_condition(self, job_id: str) -> str:
        return job_needs(self.jobs[job_id]) and self.jobs[job_id]

    def test_dispatch_never_publishes(self):
        # The publish flag is produced by the policy module, which only
        # sets it for a tag push — no input can flip it.
        self.assertNotIn("github.event.inputs", self.release)
        policy_step = steps_using(self.jobs["version"], "scripts/release/metadata.py")
        self.assertTrue(policy_step)
        for step in policy_step:
            script = step_run(step)
            self.assertIn("--tag", script)
            self.assertIn("--version", script)

    def test_registry_login_is_gated_on_publication(self):
        image = self.jobs["image"]
        logins = steps_using(image, "docker/login-action")
        self.assertTrue(logins)
        for step in logins:
            condition = step_condition(step)
            self.assertIn("publish", condition)

    def test_image_push_is_gated_on_publication(self):
        image = self.jobs["image"]
        pushes = [
            step
            for step in step_blocks(image)
            if "push: true" in step or "push: ${{" in step
        ]
        self.assertTrue(pushes)
        for step in pushes:
            self.assertIn("publish", step_condition(step))

    def test_github_release_is_gated_on_publication(self):
        package = self.jobs["package"]
        releases = steps_using(package, "softprops/action-gh-release")
        self.assertTrue(releases)
        for step in releases:
            self.assertIn("publish", step_condition(step))

    def test_dry_run_still_packages_and_verifies_artifacts(self):
        package = self.jobs["package"]
        script = "\n".join(step_run(step) for step in step_blocks(package))
        self.assertIn("scripts/release/package.sh", script)
        self.assertIn("scripts/release/verify.sh", script)

    def test_dry_run_uploads_verified_binary_artifacts(self):
        uploads = steps_using(self.jobs["package"], "actions/upload-artifact@")
        self.assertEqual(len(uploads), 1)
        self.assertNotIn("publish", step_condition(uploads[0]))
        self.assertIn("dist/*", uploads[0])

    def test_image_smoke_requires_health_and_cleans_container(self):
        smoke = steps_using(self.jobs["image"], "docker run")
        self.assertEqual(len(smoke), 1)
        script = step_run(smoke[0])
        self.assertIn("/healthz", script)
        self.assertIn("docker rm -f -v", script)
        self.assertNotIn("set +e", script)

    def test_dry_run_still_builds_and_smokes_the_image(self):
        image = self.jobs["image"]
        self.assertTrue(steps_using(image, "docker/build-push-action"))
        self.assertTrue(steps_using(image, "docker run"))


class TestReleaseWorkflowArtifacts(unittest.TestCase):
    def setUp(self):
        self.release = read_workflow("release.yml")
        self.jobs = job_blocks(self.release)

    def test_packaging_scripts_receive_version_and_output_dir(self):
        package = self.jobs["package"]
        script = "\n".join(step_run(step) for step in step_blocks(package))
        self.assertIn('scripts/release/package.sh "$RELEASE_VERSION" "$OUTPUT_DIR"', script)
        self.assertIn('scripts/release/verify.sh "$RELEASE_VERSION" "$OUTPUT_DIR"', script)

    def test_image_build_passes_the_version_to_the_dockerfile(self):
        image = self.jobs["image"]
        builds = steps_using(image, "docker/build-push-action")
        self.assertTrue(builds)
        for step in builds:
            self.assertIn("VERSION=", step)
            self.assertIn("needs.version.outputs.version", step)

    def test_three_linux_platforms_are_preserved(self):
        image = self.jobs["image"]
        builds = steps_using(image, "docker/build-push-action")
        multi_arch = [step for step in builds if "linux/arm64" in step]
        self.assertTrue(multi_arch)
        for step in multi_arch:
            self.assertIn("linux/amd64", step)
            self.assertIn("linux/arm/v7", step)
        supported = {"linux/amd64", "linux/arm64", "linux/arm/v7"}
        for step in builds:
            for platform in re.findall(r"linux/[a-z0-9/]+", step):
                self.assertIn(platform, supported)

    def test_prerelease_flags_come_from_the_policy_module(self):
        package = self.jobs["package"]
        releases = steps_using(package, "softprops/action-gh-release")
        for step in releases:
            self.assertIn("needs.version.outputs.prerelease", step)
            self.assertIn("needs.version.outputs.make_latest", step)

    def test_tag_policy_is_not_hardcoded_in_the_workflow(self):
        # Channels come from scripts/release/metadata.py, never from a
        # hardcoded `latest` tag directive.
        self.assertNotIn("type=raw", self.release)
        self.assertNotIn("latest=auto", self.release)


class TestReleaseWorkflowRegistryAvailability(unittest.TestCase):
    def setUp(self):
        self.release = read_workflow("release.yml")
        self.image = job_blocks(self.release)["image"]

    def test_optional_dockerhub_secrets_are_assigned_to_env(self):
        self.assertIn("DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}", self.image)
        self.assertIn("DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}", self.image)

    def test_the_same_availability_flag_controls_target_and_login(self):
        detect = [
            step
            for step in steps_using(self.image, "DOCKERHUB_USERNAME")
            if step_run(step)
        ]
        self.assertTrue(detect)
        for step in detect:
            self.assertIn("GITHUB_OUTPUT", step_run(step))
        login = steps_using(self.image, "docker/login-action")
        dockerhub_login = [step for step in login if "docker.io" in step]
        self.assertTrue(dockerhub_login)
        for step in dockerhub_login:
            self.assertIn("available", step_condition(step))


class TestReusableWorkflowWiring(unittest.TestCase):
    def setUp(self):
        self.pr = read_workflow("pr.yml")
        self.security = read_workflow("security.yml")

    def test_pr_workflow_runs_the_release_policy_tests(self):
        self.assertIn("scripts/release/test_workflow.py", self.pr)

    def test_checkouts_use_the_exact_triggering_sha(self):
        for name, text in (("pr.yml", self.pr), ("security.yml", self.security)):
            with self.subTest(workflow=name):
                self.assertIn("ref: ${{ github.sha }}", text)

    def test_reusable_workflows_keep_their_own_triggers(self):
        self.assertIn("pull_request:", self.pr)
        self.assertIn("push:", self.pr)
        self.assertIn("pull_request:", self.security)
        self.assertIn("schedule:", self.security)

    def test_reusable_workflows_have_distinct_concurrency_groups(self):
        def group(text: str) -> str:
            match = re.search(r"^concurrency:\n(?:  .*\n)*?  group: (.*)$", text, re.MULTILINE)
            return match.group(1).strip() if match else ""

        self.assertTrue(group(self.pr))
        self.assertTrue(group(self.security))
        self.assertNotEqual(group(self.pr), group(self.security))


class TestPriorBehaviourIsPreserved(unittest.TestCase):
    """The changed workflows must keep their previous behaviour."""

    BASE_PR_JOBS = ["backend", "frontend", "openapi", "lint"]
    BASE_SECURITY_JOBS = ["govulncheck", "npm-audit", "trivy"]
    BASE_RELEASE_ACTIONS = [
        "actions/checkout",
        "actions/setup-go",
        "actions/setup-node",
        "docker/login-action",
        "docker/build-push-action",
        "softprops/action-gh-release",
    ]

    def test_pr_workflow_keeps_its_jobs(self):
        jobs = job_blocks(read_workflow("pr.yml"))
        for job in self.BASE_PR_JOBS:
            self.assertIn(job, jobs)

    def test_security_workflow_keeps_its_jobs(self):
        jobs = job_blocks(read_workflow("security.yml"))
        for job in self.BASE_SECURITY_JOBS:
            self.assertIn(job, jobs)

    def test_release_workflow_keeps_its_actions(self):
        release = read_workflow("release.yml")
        for action in self.BASE_RELEASE_ACTIONS:
            self.assertIn(f"{action}@", release)

    def test_release_workflow_keeps_the_install_url_smoke_job(self):
        release = read_workflow("release.yml")
        self.assertIn("smoke-install-url:", release)
        self.assertIn("get.nrcc.dev/install.sh", release)

    def test_every_action_stays_pinned_to_a_commit_sha(self):
        for name in ("release.yml", "pr.yml", "security.yml"):
            text = read_workflow(name)
            for action, ref in re.findall(r"uses:\s*([^\s@]+)@([^\s#\s]+)", text):
                with self.subTest(workflow=name, action=action):
                    self.assertRegex(ref, r"^[0-9a-f]{40}$")


if __name__ == "__main__":
    unittest.main()
