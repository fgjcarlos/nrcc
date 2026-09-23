package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test slice 3 of #768 — the language-policy CI gate is a YAML
// workflow. We assert structure and required fields so a future
// refactor of the workflow keeps the contract with the scanner
// intact (scanner invocation, output format, advisory semantics).

// repoRoot is the workspace root the workflow tests walk up to.
// Go tests run with cwd = package directory; the workflow lives at
// the repo root two levels up.
const repoRoot = "../../.."

func TestLanguagePolicyWorkflow_Present(t *testing.T) {
	path := filepath.Join(repoRoot, ".github", "workflows", "language-policy.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist, got %v", path, err)
	}
}

func TestLanguagePolicyWorkflow_AdvisoryName(t *testing.T) {
	body := readWorkflow(t)
	if !strings.Contains(body, "name: Language policy") {
		t.Errorf("workflow name should include 'Language policy', got:\n%s", body)
	}
}

func TestLanguagePolicyWorkflow_Triggers(t *testing.T) {
	body := readWorkflow(t)
	// Must trigger on pull_request and push to main so the scanner
	// reports findings both at PR time and on main after merge.
	if !strings.Contains(body, "pull_request:") {
		t.Errorf("workflow must trigger on pull_request")
	}
	if !strings.Contains(body, "push:") || !strings.Contains(body, "branches: [main]") {
		t.Errorf("workflow must trigger on push to main")
	}
}

func TestLanguagePolicyWorkflow_AdvisoryPermissions(t *testing.T) {
	body := readWorkflow(t)
	// Advisory means: contents read; pull-requests write so the job can
	// post a comment when findings exist. No write access to repo
	// contents — the workflow never edits files.
	if !strings.Contains(body, "contents: read") {
		t.Errorf("workflow should declare contents: read")
	}
	if !strings.Contains(body, "pull-requests: write") {
		t.Errorf("workflow should declare pull-requests: write to post advisory comments")
	}
	if strings.Contains(body, "contents: write") {
		t.Errorf("workflow should not request contents: write (advisory only)")
	}
}

func TestLanguagePolicyWorkflow_RunsScanner(t *testing.T) {
	body := readWorkflow(t)
	// The scanner invocation is the single source of truth for the
	// policy gate. The step must use `go run ./tools/langscan` so the
	// CI runs the exact binary the scanner tests exercise.
	if !strings.Contains(body, "go run ./tools/langscan") {
		t.Errorf("workflow must run the scanner via `go run ./tools/langscan`")
	}
}

func TestLanguagePolicyWorkflow_AdvisoryExit(t *testing.T) {
	body := readWorkflow(t)
	// Advisory semantics: the scanner step must not block the job
	// when findings exist. We assert either an `|| true` or a
	// `continue-on-error: true` on the scanner step.
	scannerIdx := strings.Index(body, "go run ./tools/langscan")
	if scannerIdx < 0 {
		t.Fatalf("scanner invocation not found")
	}
	// Look at the preceding `run:` block for the scanner step.
	runStart := strings.LastIndex(body[:scannerIdx], "- name:")
	if runStart < 0 {
		t.Fatalf("scanner step header not found")
	}
	runBlock := body[runStart:scannerIdx+len("go run ./tools/langscan")+50]
	if !strings.Contains(runBlock, "|| true") && !strings.Contains(runBlock, "continue-on-error: true") {
		t.Errorf("scanner step must be advisory (|| true or continue-on-error: true), got:\n%s", runBlock)
	}
}

func TestLanguagePolicyWorkflow_PostsCommentOnPR(t *testing.T) {
	body := readWorkflow(t)
	// On PR runs, post a comment with the report. Use either
	// github-script or the gh CLI; both are acceptable as long as
	// the step is gated on the pull_request event.
	hasGhScript := strings.Contains(body, "actions/github-script")
	hasGhCLI := strings.Contains(body, "gh api") || strings.Contains(body, "gh pr comment")
	if !hasGhScript && !hasGhCLI {
		t.Errorf("workflow should post a PR comment via github-script or gh CLI")
	}
	if !strings.Contains(body, "if: github.event_name == 'pull_request'") &&
		!strings.Contains(body, `if: github.event_name == 'pull_request'`) {
		// Comment step must be gated to PR runs.
		t.Errorf("PR comment step should be gated on github.event_name == 'pull_request'")
	}
}

func readWorkflow(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "language-policy.yml"))
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	return string(b)
}
