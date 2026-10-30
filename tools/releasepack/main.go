// Command releasepack plans, builds and verifies the nrcc release
// artifacts.
//
// The tool exists so that the release pipeline and the local dry run share
// one implementation of the release contract described in issue #884:
//
//   - the set of supported native build targets (Linux only, see ADR 0004),
//   - the publication channels and their Docker tags (stable versus
//     prerelease),
//   - the checksum layout published next to every binary,
//   - the required-check gate that a publication must pass.
//
// The tool never talks to a registry and never publishes anything by
// itself: it only computes, builds and verifies local artifacts, and it
// evaluates check results that another step has already downloaded.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "releasepack: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("a subcommand is required")
	}

	switch args[0] {
	case "plan":
		return runPlan(args[1:])
	case "build":
		return runBuild(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `releasepack — nrcc release packaging helper (issue #884)

Usage:
  releasepack plan   --version <version> [--image <ref>]... [--revision <rev>]
                      [--created <rfc3339>] [--format json|github]
  releasepack build  --version <version> [--dist <dir>] [--source <dir>]
  releasepack verify --version <version> [--dist <dir>]

Subcommands:
  plan    Compute the release plan: supported targets, publication channel,
          Docker tags, image targets and OCI labels.
  build   Cross-compile the supported Linux binaries into the dist directory
          and write the SHA256SUMS file plus the per-binary sidecars.
  verify  Verify the produced artifacts: expected file set, checksums,
          sidecars and stamped version metadata.
`)
}
