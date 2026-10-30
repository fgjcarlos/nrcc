package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateGate_AcceptsSuccessfulChecks(t *testing.T) {
	runs := []checkRun{
		{Name: "Backend (Go 1.26.9)", Status: "completed", Conclusion: "success"},
		{Name: "Frontend (Node 22)", Status: "completed", Conclusion: "success"},
	}

	results := evaluateGate([]string{"Backend (Go 1.26.9)", "Frontend (Node 22)"}, runs, nil)

	for _, result := range results {
		if !result.Passed {
			t.Fatalf("expected %q to pass, got %q", result.Name, result.Detail)
		}
	}
}

func TestEvaluateGate_RejectsMissingCheck(t *testing.T) {
	results := evaluateGate([]string{"Backend (Go 1.26.9)"}, nil, nil)

	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Passed {
		t.Fatal("expected a missing check to fail the gate")
	}
	if !strings.Contains(results[0].Detail, "no check run") {
		t.Fatalf("expected a missing check detail, got %q", results[0].Detail)
	}
}

func TestEvaluateGate_RejectsFailedCheck(t *testing.T) {
	runs := []checkRun{
		{Name: "Lint (golangci-lint)", Status: "completed", Conclusion: "failure"},
	}

	results := evaluateGate([]string{"Lint (golangci-lint)"}, runs, nil)

	if results[0].Passed {
		t.Fatal("expected a failed check to fail the gate")
	}
	if !strings.Contains(results[0].Detail, "failure") {
		t.Fatalf("expected the conclusion to be reported, got %q", results[0].Detail)
	}
}

func TestEvaluateGate_RejectsIncompleteCheck(t *testing.T) {
	runs := []checkRun{
		{Name: "Docker stack acceptance (#434)", Status: "in_progress", Conclusion: ""},
	}

	results := evaluateGate([]string{"Docker stack acceptance (#434)"}, runs, nil)

	if results[0].Passed {
		t.Fatal("expected an unfinished check to fail the gate")
	}
	if !strings.Contains(results[0].Detail, "in_progress") {
		t.Fatalf("expected the status to be reported, got %q", results[0].Detail)
	}
}

func TestEvaluateGate_AcceptsSkippedCheck(t *testing.T) {
	runs := []checkRun{
		{Name: "pnpm audit", Status: "completed", Conclusion: "skipped"},
	}

	results := evaluateGate([]string{"pnpm audit"}, runs, nil)

	if !results[0].Passed {
		t.Fatalf("expected a skipped check to pass, got %q", results[0].Detail)
	}
}

func TestEvaluateGate_RequiresEveryMatrixRunToPass(t *testing.T) {
	runs := []checkRun{
		{Name: "Backend (Go 1.26.9)", Status: "completed", Conclusion: "success"},
		{Name: "Backend (Go 1.26.9)", Status: "completed", Conclusion: "failure"},
	}

	results := evaluateGate([]string{"Backend (Go 1.26.9)"}, runs, nil)

	if results[0].Passed {
		t.Fatal("expected one failing matrix run to fail the gate")
	}
}

func TestEvaluateGate_FallsBackToCommitStatus(t *testing.T) {
	statuses := []statusContext{
		{Context: "legacy-check", State: "success"},
	}

	results := evaluateGate([]string{"legacy-check"}, nil, statuses)

	if !results[0].Passed {
		t.Fatalf("expected the commit status to satisfy the gate, got %q", results[0].Detail)
	}
}

func TestRunGate_ReportsMissingRequiredCheck(t *testing.T) {
	dir := t.TempDir()
	checksPath := filepath.Join(dir, "checks.json")
	content := `{"total_count":1,"check_runs":[{"name":"Backend (Go 1.26.9)","status":"completed","conclusion":"success"}]}`
	if err := os.WriteFile(checksPath, []byte(content), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := runGate([]string{"--required", "Backend (Go 1.26.9),Frontend (Node 22)", "--checks", checksPath})
	if err == nil {
		t.Fatal("expected the gate to fail for the missing check")
	}
	if !strings.Contains(err.Error(), "1 of 2 required checks") {
		t.Fatalf("expected the failure count to be reported, got: %v", err)
	}
}

func TestRunGate_AcceptsSatisfiedRequiredChecks(t *testing.T) {
	dir := t.TempDir()
	checksPath := filepath.Join(dir, "checks.json")
	content := `{"total_count":1,"check_runs":[{"name":"Backend (Go 1.26.9)","status":"completed","conclusion":"success"}]}`
	if err := os.WriteFile(checksPath, []byte(content), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := runGate([]string{"--required", "Backend (Go 1.26.9)", "--checks", checksPath}); err != nil {
		t.Fatalf("expected the gate to pass, got: %v", err)
	}
}

func TestSplitList(t *testing.T) {
	names := splitList(" a , b ,, c ")

	expected := []string{"a", "b", "c"}
	if len(names) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
	for i, name := range expected {
		if names[i] != name {
			t.Fatalf("expected %v, got %v", expected, names)
		}
	}
}

func TestEvaluateGate_AcceptsRerunOfAFailedCheck(t *testing.T) {
	runs := []checkRun{
		{Name: "Lint (golangci-lint)", Status: "completed", Conclusion: "failure", RunAttempt: 1},
		{Name: "Lint (golangci-lint)", Status: "completed", Conclusion: "success", RunAttempt: 2},
	}

	results := evaluateGate([]string{"Lint (golangci-lint)"}, runs, nil)

	if !results[0].Passed {
		t.Fatalf("expected the re-run to satisfy the gate, got %q", results[0].Detail)
	}
}

func TestEvaluateGate_RejectsFailedLatestAttempt(t *testing.T) {
	runs := []checkRun{
		{Name: "Lint (golangci-lint)", Status: "completed", Conclusion: "success", RunAttempt: 1},
		{Name: "Lint (golangci-lint)", Status: "completed", Conclusion: "failure", RunAttempt: 2},
	}

	results := evaluateGate([]string{"Lint (golangci-lint)"}, runs, nil)

	if results[0].Passed {
		t.Fatal("expected the latest attempt to decide the gate")
	}
}

func TestEvaluateGate_RequiresEveryMatrixLegOfTheLatestAttempt(t *testing.T) {
	runs := []checkRun{
		{Name: "Backend (Go 1.26.9)", Status: "completed", Conclusion: "success", RunAttempt: 1},
		{Name: "Backend (Go 1.26.9)", Status: "completed", Conclusion: "failure", RunAttempt: 1},
	}

	results := evaluateGate([]string{"Backend (Go 1.26.9)"}, runs, nil)

	if results[0].Passed {
		t.Fatal("expected a failing matrix leg to fail the gate")
	}
}
