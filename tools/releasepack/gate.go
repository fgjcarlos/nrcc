package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Passing check conclusions.
//
// "skipped" and "neutral" count as passing because GitHub treats them as
// satisfied for required status checks: a job that is skipped (for example
// by the fork guard) must not block a release forever.
var passingConclusions = map[string]bool{
	"success": true,
	"skipped": true,
	"neutral": true,
	"":        true,
	"pending": false,
}

// checkRun mirrors the fields of a GitHub check run that the gate needs.
type checkRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	App        struct {
		Slug string `json:"slug"`
	} `json:"app"`
}

// checkRunsResponse is the payload of the GitHub "check runs for a commit"
// API endpoint.
type checkRunsResponse struct {
	CheckRuns []checkRun `json:"check_runs"`
}

// statusContext mirrors a legacy commit status context.
type statusContext struct {
	Context string `json:"context"`
	State   string `json:"state"`
}

// combinedStatusResponse is the payload of the GitHub "combined status for
// a commit" API endpoint.
type combinedStatusResponse struct {
	Statuses []statusContext `json:"statuses"`
}

// gateResult is the outcome of one required check.
type gateResult struct {
	Name   string
	Passed bool
	Detail string
}

// runGate implements the "gate" subcommand.
func runGate(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	required := fs.String("required", "", "comma separated list of required check names")
	checksPath := fs.String("checks", "", "path to the check runs JSON downloaded from GitHub")
	statusesPath := fs.String("statuses", "", "path to the combined commit status JSON downloaded from GitHub")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*required) == "" {
		return errors.New("--required is required")
	}

	names := splitList(*required)

	runs, err := loadCheckRuns(*checksPath)
	if err != nil {
		return err
	}

	statuses, err := loadStatuses(*statusesPath)
	if err != nil {
		return err
	}

	results := evaluateGate(names, runs, statuses)

	failures := 0
	for _, result := range results {
		if result.Passed {
			fmt.Printf("required check %q: ok (%s)\n", result.Name, result.Detail)
			continue
		}
		failures++
		fmt.Printf("required check %q: FAILED (%s)\n", result.Name, result.Detail)
	}

	if failures > 0 {
		return fmt.Errorf("%d of %d required checks did not succeed for the release commit", failures, len(results))
	}

	fmt.Printf("all %d required checks succeeded\n", len(results))
	return nil
}

// splitList splits a comma separated list and drops empty entries.
func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		names = append(names, trimmed)
	}
	return names
}

// loadCheckRuns reads the check runs JSON file.
func loadCheckRuns(path string) ([]checkRun, error) {
	if path == "" {
		return nil, nil
	}

	// #nosec G304 -- the path is provided by the operator of the release
	// pipeline, not by request input.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read check runs file: %w", err)
	}

	var response checkRunsResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, fmt.Errorf("parse check runs file: %w", err)
	}

	return response.CheckRuns, nil
}

// loadStatuses reads the combined commit status JSON file.
func loadStatuses(path string) ([]statusContext, error) {
	if path == "" {
		return nil, nil
	}

	// #nosec G304 -- the path is provided by the operator of the release
	// pipeline, not by request input.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read combined status file: %w", err)
	}

	var response combinedStatusResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, fmt.Errorf("parse combined status file: %w", err)
	}

	return response.Statuses, nil
}

// evaluateGate evaluates every required check name against the downloaded
// check results.
func evaluateGate(names []string, runs []checkRun, statuses []statusContext) []gateResult {
	results := make([]gateResult, 0, len(names))

	for _, name := range names {
		results = append(results, evaluateCheck(name, runs, statuses))
	}

	return results
}

// evaluateCheck evaluates one required check name.
func evaluateCheck(name string, runs []checkRun, statuses []statusContext) gateResult {
	matches := matchingRuns(name, runs)
	if len(matches) == 0 {
		return evaluateStatus(name, statuses)
	}

	for _, run := range matches {
		if run.Status != "" && run.Status != "completed" {
			return gateResult{Name: name, Passed: false, Detail: fmt.Sprintf("check is %s", run.Status)}
		}
		if !passingConclusions[run.Conclusion] {
			return gateResult{Name: name, Passed: false, Detail: fmt.Sprintf("conclusion is %q", run.Conclusion)}
		}
	}

	detail := "success"
	if len(matches) > 1 {
		detail = fmt.Sprintf("success (%d check runs)", len(matches))
	}
	return gateResult{Name: name, Passed: true, Detail: detail}
}

// matchingRuns returns the check runs that carry the given name.
func matchingRuns(name string, runs []checkRun) []checkRun {
	var matches []checkRun
	for _, run := range runs {
		if run.Name == name {
			matches = append(matches, run)
		}
	}
	return matches
}

// evaluateStatus falls back to the legacy commit status contexts when no
// check run carries the required name.
func evaluateStatus(name string, statuses []statusContext) gateResult {
	for _, status := range statuses {
		if status.Context != name {
			continue
		}
		if status.State == "success" {
			return gateResult{Name: name, Passed: true, Detail: "success (commit status)"}
		}
		return gateResult{Name: name, Passed: false, Detail: fmt.Sprintf("commit status is %q", status.State)}
	}

	return gateResult{Name: name, Passed: false, Detail: "no check run or commit status found for the release commit"}
}
