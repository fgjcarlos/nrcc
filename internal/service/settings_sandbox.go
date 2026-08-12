package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Sentinel errors returned by ParseAdminAuthViaSandbox. Callers should
// distinguish missing/blocked/running-error conditions via errors.Is.
var (
	ErrAdminAuthMissing = errors.New("settings.js: adminAuth block not found")
	ErrSandboxRuntime   = errors.New("settings.js: sandbox runtime error")
	ErrSandboxSyntax    = errors.New("settings.js: syntax error in sandbox")
)

// ParseAdminAuthViaSandbox extracts the adminAuth block from a settings.js
// content string using a goja-backed JavaScript sandbox. Compared to the
// legacy regex parser this implementation:
//
//   - correctly handles escaped quotes (e.g. password: "a\"b")
//   - ignores comments (line `// ...` and block `/* ... */`) without false matches
//   - reads whatever the script actually exported, not a regex guess
//   - rejects forbidden host APIs (require, process, Buffer, etc.) explicitly
//
// On missing adminAuth (e.g. the file does not export one) the function
// returns ErrAdminAuthMissing. The caller decides whether that is fatal.
func ParseAdminAuthViaSandbox(content string) (*model.AdminAuth, error) {
	if content == "" {
		return nil, ErrAdminAuthMissing
	}

	// Wrap content with module.exports hooks so the user content can either
	// declare `module.exports = { adminAuth: ... }` or assign `exports.adminAuth = ...`.
	wrapped := "var module = { exports: {} };\n" +
		"var exports = module.exports;\n" +
		content + "\n"

	rt := goja.New()

	// Block host APIs explicitly. Setting to goja.Undefined() means any
	// access throws a TypeError when the script tries to use them.
	for _, name := range []string{
		"require", "process", "Buffer", "globalThis", "global",
		"setTimeout", "setInterval", "fetch",
	} {
		// Per goja docs Set may return an error for some value kinds; we
		// pass goja.Undefined() which is always representable, but we still
		// capture and surface the error if it ever happens.
		if err := rt.Set(name, goja.Undefined()); err != nil {
			return nil, fmt.Errorf("sandbox setup: block %q: %w", name, err)
		}
	}

	if _, err := rt.RunString(wrapped); err != nil {
		// Map goja syntax errors to a sentinel; everything else as runtime.
		if isGojaSyntaxError(err) {
			return nil, fmt.Errorf("%w: %w", ErrSandboxSyntax, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrSandboxRuntime, err)
	}

	return extractAdminAuth(rt)
}

// extractAdminAuth reads `module.exports.adminAuth` from the sandbox runtime
// and converts the JS value into a model.AdminAuth. Returns ErrAdminAuthMissing
// if the key is absent or null.
func extractAdminAuth(rt *goja.Runtime) (*model.AdminAuth, error) {
	moduleVal := rt.Get("module")
	if moduleVal == nil || goja.IsUndefined(moduleVal) || goja.IsNull(moduleVal) {
		return nil, ErrAdminAuthMissing
	}
	moduleObj := moduleVal.ToObject(rt)
	exportsVal := moduleObj.Get("exports")
	if exportsVal == nil || goja.IsUndefined(exportsVal) || goja.IsNull(exportsVal) {
		return nil, ErrAdminAuthMissing
	}
	exportsObj := exportsVal.ToObject(rt)
	adminAuthVal := exportsObj.Get("adminAuth")
	if adminAuthVal == nil || goja.IsUndefined(adminAuthVal) || goja.IsNull(adminAuthVal) {
		return nil, ErrAdminAuthMissing
	}

	adminAuthObj := adminAuthVal.ToObject(rt)

	auth := &model.AdminAuth{}
	if typeVal := adminAuthObj.Get("type"); typeVal != nil && !goja.IsUndefined(typeVal) && !goja.IsNull(typeVal) {
		auth.Type = typeVal.String()
	}
	if auth.Type == "" {
		return nil, fmt.Errorf("%w: missing type", ErrAdminAuthMissing)
	}

	usersVal := adminAuthObj.Get("users")
	if usersVal == nil || goja.IsUndefined(usersVal) || goja.IsNull(usersVal) {
		return auth, nil
	}
	usersObj := usersVal.ToObject(rt)
	usersKeys := usersObj.Keys()
	for _, key := range usersKeys {
		userVal := usersObj.Get(key)
		if userVal == nil || goja.IsUndefined(userVal) || goja.IsNull(userVal) {
			continue
		}
		userObj := userVal.ToObject(rt)
		username := readStringProp(userObj, "username")
		password := readStringProp(userObj, "password")
		if username == "" || password == "" {
			continue
		}
		auth.Users = append(auth.Users, model.AdminAuthUser{
			Username:    username,
			Password:    password,
			Permissions: readStringProp(userObj, "permissions"),
		})
	}
	return auth, nil
}

// readStringProp reads a JS object property as a string. Returns "" for
// undefined/null/missing/non-string values.
func readStringProp(obj *goja.Object, name string) string {
	v := obj.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	return v.String()
}

// isGojaSyntaxError reports whether err originates from a JavaScript parse
// failure (vs a runtime TypeError or ReferenceError). goja doesn't expose a
// dedicated error type for syntax errors, so we match on the message prefix.
func isGojaSyntaxError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, marker := range []string{"SyntaxError", "Unexpected token", "Expected"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
