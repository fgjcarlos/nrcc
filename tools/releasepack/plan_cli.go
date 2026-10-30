package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// imageList is a repeatable string flag.
type imageList []string

func (l *imageList) String() string {
	return strings.Join(*l, ",")
}

func (l *imageList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("image reference is empty")
	}
	*l = append(*l, value)
	return nil
}

// runPlan implements the "plan" subcommand.
func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	version := fs.String("version", "", "release version, for example v1.2.3 or v1.2.3-rc.1")
	revision := fs.String("revision", "", "commit revision stamped into the image labels")
	created := fs.String("created", "", "creation timestamp (RFC 3339) stamped into the image labels")
	format := fs.String("format", "json", "output format: json or github")
	var images imageList
	fs.Var(&images, "image", "image reference to publish (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" {
		return errors.New("--version is required")
	}

	parsed, err := parseVersion(*version)
	if err != nil {
		return err
	}

	plan := newPlan(parsed, images, *revision, *created)

	switch *format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(plan)
	case "github":
		return writeGitHubOutput(os.Stdout, plan)
	default:
		return fmt.Errorf("unknown --format %q", *format)
	}
}

// writeGitHubOutput writes the plan as GitHub Actions output entries.
//
// Multi-line values use the heredoc form so the entries can be appended to
// the GITHUB_OUTPUT file verbatim.
func writeGitHubOutput(out *os.File, plan Plan) error {
	labels, err := json.Marshal(plan.Labels)
	if err != nil {
		return fmt.Errorf("encode labels: %w", err)
	}

	entries := []struct {
		key   string
		value string
	}{
		{key: "version", value: plan.Version},
		{key: "tag", value: plan.Tag},
		{key: "channel", value: plan.Channel},
		{key: "prerelease", value: strconv.FormatBool(plan.Prerelease)},
		{key: "tags", value: strings.Join(plan.Tags, "\n")},
		{key: "metadata_tags", value: strings.Join(plan.MetadataTags, "\n")},
		{key: "labels", value: string(labels)},
	}

	for _, entry := range entries {
		if strings.Contains(entry.value, "\n") {
			if _, err := fmt.Fprintf(out, "%s<<EOF\n%s\nEOF\n", entry.key, entry.value); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(out, "%s=%s\n", entry.key, entry.value); err != nil {
			return err
		}
	}

	images := strings.Join(plan.Images, "\n")
	if _, err := fmt.Fprintf(out, "images<<EOF\n%s\nEOF\n", images); err != nil {
		return err
	}

	targets, err := json.Marshal(plan.Targets)
	if err != nil {
		return fmt.Errorf("encode targets: %w", err)
	}
	if _, err := fmt.Fprintf(out, "targets=%s\n", string(targets)); err != nil {
		return err
	}

	return nil
}
