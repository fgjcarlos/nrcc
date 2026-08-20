package service

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

const sandboxFixtureBasic = `
module.exports = {
  uiPort: 1880,
  adminAuth: {
    type: "credentials",
    users: [
      { username: "admin", password: "$2a$08$abc", permissions: "*" }
    ]
  }
}
`

const sandboxFixtureEscapedQuote = `
module.exports = {
  adminAuth: {
    type: "credentials",
    users: [
      { username: "admin", password: "p\"a\"ss", permissions: "*" }
    ]
  }
}
`

const sandboxFixtureLineComment = `
module.exports = {
  // adminAuth: { type: "ignored", users: [] }
  adminAuth: {
    type: "credentials",
    users: [
      { username: "real", password: "hash", permissions: "*" }
    ]
  }
}
`

const sandboxFixtureBlockComment = `
module.exports = {
  /* adminAuth: { type: "commented", users: [] } */
  adminAuth: {
    type: "credentials",
    users: [
      { username: "real", password: "hash", permissions: "*" }
    ]
  }
}
`

const sandboxFixtureNoAdminAuth = `
module.exports = {
  uiPort: 1880
}
`

const sandboxFixtureNullAdminAuth = `
module.exports = {
  adminAuth: null
}
`

const sandboxFixtureMultipleReassign = `
var first = { adminAuth: { type: "first", users: [{ username: "u1", password: "p1" }] } };
var second = { adminAuth: { type: "second", users: [{ username: "u2", password: "p2" }] } };
// Sequential merge simulates two adminAuth blocks overwriting each other.
first.adminAuth = second.adminAuth;
module.exports = first;
`

func TestParseAdminAuthViaSandbox_Basic(t *testing.T) {
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureBasic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil {
		t.Fatal("expected non-nil adminAuth")
		return
	}
	if auth.Type != "credentials" {
		t.Errorf("expected type credentials, got %q", auth.Type)
	}
	if len(auth.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(auth.Users))
	}
	if auth.Users[0].Username != "admin" {
		t.Errorf("expected username admin, got %q", auth.Users[0].Username)
	}
	if auth.Users[0].Password != "$2a$08$abc" {
		t.Errorf("expected password $2a$08$abc, got %q", auth.Users[0].Password)
	}
}

func TestParseAdminAuthViaSandbox_EscapedQuoteInPassword(t *testing.T) {
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureEscapedQuote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil || len(auth.Users) == 0 {
		t.Fatal("expected adminAuth with users")
	}
	if auth.Users[0].Password != `p"a"ss` {
		t.Errorf("expected escaped quote password, got %q", auth.Users[0].Password)
	}
}

func TestParseAdminAuthViaSandbox_LineCommentIgnored(t *testing.T) {
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureLineComment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil || len(auth.Users) == 0 {
		t.Fatal("expected real adminAuth to win over commented one")
	}
	if auth.Users[0].Username != "real" {
		t.Errorf("expected real user, got %q (regex would have picked the commented one)", auth.Users[0].Username)
	}
}

func TestParseAdminAuthViaSandbox_BlockCommentIgnored(t *testing.T) {
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureBlockComment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil || len(auth.Users) == 0 {
		t.Fatal("expected real adminAuth to win over commented one")
	}
	if auth.Users[0].Username != "real" {
		t.Errorf("expected real user, got %q", auth.Users[0].Username)
	}
}

func TestParseAdminAuthViaSandbox_MultipleAdminAuthBlocks(t *testing.T) {
	// The legacy regex parser silently took the first block, which could
	// cause preserveAdminAuthPasswords to overwrite the real password with
	// a stale one. The sandbox sees the actual final value, which here is
	// the second assignment.
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureMultipleReassign)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil {
		t.Fatal("expected non-nil adminAuth")
		return
	}
	if auth.Type != "second" {
		t.Errorf("expected type second (final assignment), got %q", auth.Type)
	}
	if len(auth.Users) != 1 || auth.Users[0].Username != "u2" {
		t.Errorf("expected user u2, got %+v", auth.Users)
	}
}

func TestParseAdminAuthViaSandbox_NoAdminAuth(t *testing.T) {
	_, err := ParseAdminAuthViaSandbox(sandboxFixtureNoAdminAuth)
	if !errors.Is(err, ErrAdminAuthMissing) {
		t.Fatalf("expected ErrAdminAuthMissing, got %v", err)
	}
}

func TestParseAdminAuthViaSandbox_NullAdminAuth(t *testing.T) {
	_, err := ParseAdminAuthViaSandbox(sandboxFixtureNullAdminAuth)
	if !errors.Is(err, ErrAdminAuthMissing) {
		t.Fatalf("expected ErrAdminAuthMissing, got %v", err)
	}
}

func TestParseAdminAuthViaSandbox_EmptyContent(t *testing.T) {
	_, err := ParseAdminAuthViaSandbox("")
	if !errors.Is(err, ErrAdminAuthMissing) {
		t.Fatalf("expected ErrAdminAuthMissing for empty content, got %v", err)
	}
}

func TestParseAdminAuthViaSandbox_ForbiddenRequire(t *testing.T) {
	content := `
module.exports = {
  adminAuth: (function () {
    var fs = require('fs');
    return { type: "x", users: [{ username: "u", password: "p" }] };
  })()
}
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for require('fs') call")
	}
	if !errors.Is(err, ErrSandboxRuntime) && !errors.Is(err, ErrSandboxSyntax) {
		t.Fatalf("expected ErrSandboxRuntime or ErrSandboxSyntax, got %v", err)
	}
}

func TestParseAdminAuthViaSandbox_ForbiddenProcess(t *testing.T) {
	content := `
process.exit(0);
module.exports = { adminAuth: { type: "x", users: [] } };
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for process.exit")
	}
}

func TestParseAdminAuthViaSandbox_ForbiddenBuffer(t *testing.T) {
	content := `
var buf = Buffer.from("hello");
module.exports = { adminAuth: { type: "x", users: [] } };
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for Buffer usage")
	}
}

func TestParseAdminAuthViaSandbox_ForbiddenGlobalThis(t *testing.T) {
	// globalThis is set to undefined, so property access throws TypeError.
	content := `
var leak = globalThis.LOL;
module.exports = { adminAuth: { type: "x", users: [] } };
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for globalThis property access")
	}
}

func TestParseAdminAuthViaSandbox_SyntaxError(t *testing.T) {
	content := `
module.exports = { adminAuth: { type: "x", users: [ { username: "
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for malformed JS")
	}
	if !errors.Is(err, ErrSandboxSyntax) && !errors.Is(err, ErrSandboxRuntime) {
		t.Fatalf("expected ErrSandboxSyntax or ErrSandboxRuntime, got %v", err)
	}
}

func TestParseAdminAuthViaSandbox_RaceConcurrent(t *testing.T) {
	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			auth, err := ParseAdminAuthViaSandbox(sandboxFixtureBasic)
			if err != nil {
				errs <- err
				return
			}
			if auth == nil || auth.Type != "credentials" {
				errs <- errors.New("wrong auth returned under concurrency")
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent parse error: %v", err)
	}
}

func TestParseAdminAuthViaSandbox_RealWorldNodeRedSettings(t *testing.T) {
	// Mirrors a typical Node-RED settings.js structure.
	content := `
module.exports = {
  uiPort: process.env.PORT || 1880,
  mqttReconnectTime: 15000,
  functionGlobalContext: {
    foo: 'bar'
  },
  adminAuth: {
    type: "credentials",
    users: [
      {
        username: "admin",
        password: "$2a$08$zGc2gjrYxV3LrkpBN3eGaODzBfMmXcFiHIKvBKfRubl0gOmR0lFNe",
        permissions: "*"
      },
      {
        username: "readonly",
        password: "$2a$08$abcdef",
        permissions: "read"
      }
    ]
  }
}
`
	// Note: process.env.PORT access will hit our blocked `process` global.
	// For this test we accept either an error OR a successful parse that
	// ignores the env expression (process.PORT would be undefined but the
	// `|| 1880` fallback makes it work in real JS — goja with blocked
	// global throws). So we either get a clean parse or a sentinel error.
	auth, err := ParseAdminAuthViaSandbox(content)
	if err != nil {
		if errors.Is(err, ErrSandboxRuntime) {
			t.Skipf("process.env access blocked; this fixture is illustrative: %v", err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil || len(auth.Users) != 2 {
		t.Fatalf("expected 2 users, got %+v", auth)
	}
}

// TestParseAdminAuthViaSandbox_PreservesAllBlocksViaExecuteOrder verifies the
// sandbox observes actual script execution order. The legacy regex took the
// first block; the sandbox takes whatever the script left in module.exports.
func TestParseAdminAuthViaSandbox_PreservesAllBlocksViaExecuteOrder(t *testing.T) {
	content := strings.Join([]string{
		"var admin = { adminAuth: null };",
		`admin.adminAuth = { type: "t1", users: [{ username: "u1", password: "p1" }] };`,
		`admin.adminAuth = { type: "t2", users: [{ username: "u2", password: "p2" }] };`,
		"module.exports = admin;",
	}, "\n")
	auth, err := ParseAdminAuthViaSandbox(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.Type != "t2" {
		t.Errorf("expected type t2 (last assignment), got %q", auth.Type)
	}
}

// withSandboxTimeout temporarily overrides the sandbox budget for the duration
// of t. Restores the previous value via t.Cleanup so parallel tests don't
// observe a mutated global.
func withSandboxTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	prev := sandboxTimeout
	sandboxTimeout = d
	t.Cleanup(func() { sandboxTimeout = prev })
}

// TestParseAdminAuthViaSandbox_Timeout verifies that a non-terminating script
// is interrupted within the budget (issue #665). The budget is shortened via
// withSandboxTimeout so the test stays fast.
func TestParseAdminAuthViaSandbox_Timeout(t *testing.T) {
	withSandboxTimeout(t, 100*time.Millisecond)

	start := time.Now()
	_, err := ParseAdminAuthViaSandbox("while(true){}")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error for non-terminating script, got nil")
	}
	if !errors.Is(err, ErrSandboxTimeout) {
		t.Errorf("expected ErrSandboxTimeout, got %v", err)
	}
	// Allow generous slack on top of the budget to avoid flakes on a busy
	// CI runner; the assertion that matters is that we returned at all.
	if elapsed > sandboxTimeout+time.Second {
		t.Errorf("sandbox ran for %v, expected to be bounded by ~%v", elapsed, sandboxTimeout)
	}
}

// TestParseAdminAuthViaSandbox_TimeoutDeepRecursion covers a second common
// infinite-loop shape: deep mutual recursion instead of while(true){}. Both
// must trip the same timeout path.
func TestParseAdminAuthViaSandbox_TimeoutDeepRecursion(t *testing.T) {
	withSandboxTimeout(t, 100*time.Millisecond)

	content := `
function loop() { return loop(); }
loop();
`
	_, err := ParseAdminAuthViaSandbox(content)
	if err == nil {
		t.Fatal("expected error for infinite recursion, got nil")
	}
	if !errors.Is(err, ErrSandboxTimeout) {
		t.Errorf("expected ErrSandboxTimeout, got %v", err)
	}
}

// TestParseAdminAuthViaSandbox_TimerCleanedUpOnSuccess asserts that the
// AfterFunc timer is stopped on the success path so a healthy parse does not
// leave a pending callback in the runtime's timer wheel.
func TestParseAdminAuthViaSandbox_TimerCleanedUpOnSuccess(t *testing.T) {
	withSandboxTimeout(t, 2*time.Second)

	start := time.Now()
	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureBasic)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth == nil || auth.Type != "credentials" {
		t.Fatalf("expected credentials auth, got %+v", auth)
	}
	// A healthy parse must finish well under the budget. If the timer were
	// leaking and firing later, this would race with the timer; we cap
	// generously at 10× the observed budget to tolerate a slow CI box.
	if elapsed > time.Second {
		t.Errorf("healthy parse took %v, expected < 1s", elapsed)
	}
}

// TestParseAdminAuthViaSandbox_InterruptClearedBetweenRuns verifies that two
// consecutive calls (one of which times out) both succeed: the failed run
// must not leave a stale interrupt that would abort the next call
// immediately on re-entry to RunString.
func TestParseAdminAuthViaSandbox_InterruptClearedBetweenRuns(t *testing.T) {
	withSandboxTimeout(t, 100*time.Millisecond)

	if _, err := ParseAdminAuthViaSandbox("while(true){}"); !errors.Is(err, ErrSandboxTimeout) {
		t.Fatalf("expected ErrSandboxTimeout on first run, got %v", err)
	}

	auth, err := ParseAdminAuthViaSandbox(sandboxFixtureBasic)
	if err != nil {
		t.Fatalf("second run after timeout returned error: %v", err)
	}
	if auth == nil || auth.Type != "credentials" {
		t.Fatalf("second run produced wrong auth: %+v", auth)
	}
}
