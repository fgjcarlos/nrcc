package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Sentinel errors returned by ParseAdminAuthViaSandbox. Callers should
// distinguish missing/blocked/running-error conditions via errors.Is.
var (
	ErrAdminAuthMissing = errors.New("settings.js: adminAuth block not found")
	ErrSandboxRuntime   = errors.New("settings.js: sandbox runtime error")
	ErrSandboxSyntax    = errors.New("settings.js: syntax error in sandbox")
	ErrSandboxTimeout   = errors.New("settings.js: sandbox execution timed out")
	// ErrSandboxHostAPI distinguishes source-managed Node-RED settings from
	// inspection failures. No host API is executed or exposed to the script.
	ErrSandboxHostAPI = errors.New("settings.js: host API unavailable in sandbox")
)

// sandboxTimeout caps how long ParseAdminAuthViaSandbox is allowed to run the
// settings.js script. settings.js only declares a config object, so a healthy
// parse finishes in well under 10 ms — 2 s is generous and short enough that
// a single bad file cannot permanently exhaust goroutines via /api/config.
// It is a package var (not a const) so tests can shorten it; production code
// must not mutate it.
var sandboxTimeout = 2 * time.Second

// ParseAdminAuthViaSandbox extracts the adminAuth block from a settings.js
// content string using a goja-backed JavaScript sandbox. Compared to the
// legacy regex parser this implementation:
//
//   - correctly handles escaped quotes (e.g. password: "a\"b")
//   - ignores comments (line `// ...` and block `/* ... */`) without false matches
//   - reads whatever the script actually exported, not a regex guess
//   - rejects forbidden host APIs (require, process, Buffer, etc.) explicitly
//   - bounds execution time so a non-terminating script cannot hang the read
//     path or leak goroutines (see issue #665)
//
// On missing adminAuth (e.g. the file does not export one) the function
// returns ErrAdminAuthMissing. The caller decides whether that is fatal.
func ParseAdminAuthViaSandbox(content string) (*model.AdminAuth, error) {
	surfaces, err := ParseAuthenticationSurfacesViaSandbox(content)
	if err != nil {
		return nil, err
	}
	if surfaces.AdminAuth == nil {
		return nil, ErrAdminAuthMissing
	}
	return surfaces.AdminAuth, nil
}

// ParseAuthenticationSurfacesViaSandbox extracts the canonical Node-RED
// authentication surfaces. It recognizes only adminAuth, httpNodeAuth, and
// httpStaticAuth; legacy aliases are never migrated implicitly.
func ParseAuthenticationSurfacesViaSandbox(content string) (*model.AuthenticationSurfaces, error) {
	if content == "" {
		return nil, ErrAdminAuthMissing
	}

	// Wrap content with module.exports hooks so the user content can either
	// declare `module.exports = { adminAuth: ... }` or assign `exports.adminAuth = ...`.
	wrapped := "var module = { exports: {} };\n" +
		"var exports = module.exports;\n" +
		content + "\n"

	// Compile the entire source before inspection. A blocked host access must
	// never hide a syntax error later in a source-managed settings document.
	program, err := goja.Compile("settings.js", wrapped, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSandboxSyntax, err)
	}
	rt := goja.New()

	// Bound execution: a non-terminating script (while(true){}, infinite
	// recursion, ...) would otherwise block RunString forever and leak the
	// goroutine. After sandboxTimeout, goja's interrupt mechanism raises an
	// *InterruptedError wrapping ErrSandboxTimeout, which we map to the
	// caller-visible sentinel. The defer stops the timer and clears the
	// interrupt on every path (success, syntax error, timeout) so the runtime
	// is safe to reuse under concurrent callers.
	timeout := sandboxTimeout
	timer := time.AfterFunc(timeout, func() {
		rt.Interrupt(ErrSandboxTimeout)
	})
	defer func() {
		timer.Stop()
		rt.ClearInterrupt()
	}()

	// Block host APIs with a typed inspection error instead of an ambiguous
	// TypeError. Settings may use these APIs in Node-RED, but this inspector
	// must never call them or confuse blocked access with missing auth.
	for _, name := range []string{
		"require", "process", "Buffer", "globalThis", "global",
		"setTimeout", "setInterval", "fetch",
	} {
		blocked := rt.ToValue(func(goja.FunctionCall) goja.Value {
			panic(rt.NewGoError(fmt.Errorf("%w: %s", ErrSandboxHostAPI, name)))
		})
		if err := rt.GlobalObject().DefineAccessorProperty(name, blocked, blocked, goja.FLAG_FALSE, goja.FLAG_FALSE); err != nil {
			return nil, fmt.Errorf("sandbox setup: block %q: %w", name, err)
		}
	}

	if _, err := rt.RunProgram(program); err != nil {
		// errors.Is walks the Unwrap chain; goja's *InterruptedError.Unwrap
		// returns the value we passed to rt.Interrupt, which is ErrSandboxTimeout.
		if errors.Is(err, ErrSandboxTimeout) {
			return nil, fmt.Errorf("%w: script did not terminate within %s", ErrSandboxTimeout, timeout)
		}
		return nil, fmt.Errorf("%w: %w", ErrSandboxRuntime, err)
	}

	return extractAuthenticationSurfaces(rt)
}

func extractAuthenticationSurfaces(rt *goja.Runtime) (*model.AuthenticationSurfaces, error) {
	surfaces := &model.AuthenticationSurfaces{}
	adminAuth, err := extractAdminAuth(rt)
	if err != nil && !errors.Is(err, ErrAdminAuthMissing) {
		return nil, err
	}
	surfaces.AdminAuth = adminAuth

	exports := rt.Get("module").ToObject(rt).Get("exports")
	if exports == nil || goja.IsUndefined(exports) || goja.IsNull(exports) {
		return nil, ErrAdminAuthMissing
	}
	exportsObject := exports.ToObject(rt)
	surfaces.HTTPNodeAuth = extractHTTPBasicAuth(rt, exportsObject.Get("httpNodeAuth"))
	surfaces.HTTPStaticAuth = extractHTTPBasicAuth(rt, exportsObject.Get("httpStaticAuth"))
	if surfaces.AdminAuth == nil && surfaces.HTTPNodeAuth == nil && surfaces.HTTPStaticAuth == nil {
		return nil, ErrAdminAuthMissing
	}
	return surfaces, nil
}

func extractHTTPBasicAuth(rt *goja.Runtime, value goja.Value) *model.HTTPBasicAuth {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	object := value.ToObject(rt)
	user := readStringProp(object, "user")
	pass := readStringProp(object, "pass")
	if user == "" || pass == "" {
		return nil
	}
	return &model.HTTPBasicAuth{User: user, Pass: pass}
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
	if expiry := adminAuthObj.Get("sessionExpiryTime"); expiry != nil && !goja.IsUndefined(expiry) && !goja.IsNull(expiry) {
		auth.SessionExpiryTime = int(expiry.ToInteger())
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
