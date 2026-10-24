package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// Issue #880 — the reconciliation needs to know what the settings source
// actually declares. The regex parsers can read a value but not the exact
// set of top-level keys, and they cannot read the structured blocks
// (editorTheme, runtimeState, env) at all.
//
// exportSettingsObject evaluates the source in the isolated goja sandbox and
// returns the exported top-level object as generic Go values. The sandbox
// rules are the ones the authentication inspector already enforces: host
// APIs stay blocked, execution is bounded, and a source that cannot be
// evaluated is reported as an error instead of being partially trusted.
//
// The result is produced with the sandbox's own JSON.stringify so a value
// with no JSON representation (a function) is skipped instead of failing
// the whole export. The original stringify is captured before the source
// runs, so a source that replaces the global JSON cannot influence the
// inspection.

// reconcileBlockedGlobals are the globals the reconciliation inspector
// blocks. The list mirrors the authentication inspector's block list; the
// two inspectors stay independent so a change to one does not silently
// change the other.
var reconcileBlockedGlobals = []string{
	"require", "process", "Buffer", "globalThis", "global",
	"setTimeout", "setInterval", "fetch",
}

// exportSettingsObject evaluates the settings.js source in the sandbox and
// returns the exported top-level entries. An empty map means the source
// exports nothing; an error means the source could not be inspected.
func exportSettingsObject(content string) (map[string]any, error) {
	if content == "" {
		return nil, fmt.Errorf("%w: empty source", ErrSandboxSyntax)
	}

	wrapped := "var module = { exports: {} };\n" +
		"var exports = module.exports;\n" +
		content + "\n"
	program, err := goja.Compile("settings.js", wrapped, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSandboxSyntax, err)
	}

	rt := goja.New()
	// Capture the pristine stringify before the source runs: the source is
	// operator-authored and must not be able to influence the inspection.
	stringify, ok := goja.AssertFunction(rt.Get("JSON").ToObject(rt).Get("stringify"))
	if !ok {
		return nil, fmt.Errorf("%w: JSON.stringify is unavailable", ErrSandboxRuntime)
	}

	timeout := sandboxTimeout
	timer := time.AfterFunc(timeout, func() {
		rt.Interrupt(ErrSandboxTimeout)
	})
	defer func() {
		timer.Stop()
		rt.ClearInterrupt()
	}()

	for _, name := range reconcileBlockedGlobals {
		blocked := rt.ToValue(func(goja.FunctionCall) goja.Value {
			panic(rt.NewGoError(fmt.Errorf("%w: %s", ErrSandboxHostAPI, name)))
		})
		if err := rt.GlobalObject().DefineAccessorProperty(name, blocked, blocked, goja.FLAG_FALSE, goja.FLAG_FALSE); err != nil {
			return nil, fmt.Errorf("sandbox setup: block %q: %w", name, err)
		}
	}

	if _, err := rt.RunProgram(program); err != nil {
		if errors.Is(err, ErrSandboxTimeout) {
			return nil, fmt.Errorf("%w: script did not terminate within %s", ErrSandboxTimeout, timeout)
		}
		return nil, fmt.Errorf("%w: %w", ErrSandboxRuntime, err)
	}

	moduleVal := rt.Get("module")
	if moduleVal == nil || goja.IsUndefined(moduleVal) || goja.IsNull(moduleVal) {
		return map[string]any{}, nil
	}
	exportsVal := moduleVal.ToObject(rt).Get("exports")
	if exportsVal == nil || goja.IsUndefined(exportsVal) || goja.IsNull(exportsVal) {
		return map[string]any{}, nil
	}

	encoded, err := stringify(goja.Undefined(), exportsVal)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSandboxRuntime, err)
	}
	if encoded == nil || goja.IsUndefined(encoded) || goja.IsNull(encoded) {
		return map[string]any{}, nil
	}
	text := encoded.String()
	if text == "" {
		return map[string]any{}, nil
	}
	entries := map[string]any{}
	if err := json.Unmarshal([]byte(text), &entries); err != nil {
		return nil, fmt.Errorf("%w: exported settings are not an object: %w", ErrSandboxRuntime, err)
	}
	return entries, nil
}
