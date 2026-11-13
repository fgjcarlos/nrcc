package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrInvalidLibraryAlias reports an alias that cannot be rendered safely as a
// functionGlobalContext key.
var ErrInvalidLibraryAlias = errors.New("invalid library alias")

// validAliasRe keeps aliases to identifier / package-name characters so the
// rendered key can never break out of its JavaScript string literal.
var validAliasRe = regexp.MustCompile(`^[A-Za-z0-9_$@./-]{1,214}$`)

// globalContextOpenRe finds the top-level `functionGlobalContext: {` opener;
// commented-out lines never match because only whitespace may precede the key.
var globalContextOpenRe = regexp.MustCompile(`(?m)^[ \t]*functionGlobalContext\s*:\s*\{`)

// libraryMarker tags the functionGlobalContext lines NRCC owns, so uninstall
// removes only those and operator-authored entries stay byte-stable.
const libraryMarker = "// nrcc:library "

// libraryModuleName strips a version or range from an npm specifier:
// "lodash@4" -> "lodash", "@scope/pkg@^1" -> "@scope/pkg".
func libraryModuleName(pkg string) string {
	if i := strings.LastIndex(pkg, "@"); i > 0 {
		return pkg[:i]
	}
	return pkg
}

// resolveLibraryAlias defaults an empty alias to the module name and rejects
// aliases that are not safe to render.
func resolveLibraryAlias(pkg, alias string) (string, error) {
	// Validate the name first so a bad name reports INVALID_PACKAGE_NAME,
	// not an alias error from the defaulted alias.
	if err := ValidatePackageName(pkg); err != nil {
		return "", err
	}
	if alias == "" {
		alias = libraryModuleName(pkg)
	}
	if !validAliasRe.MatchString(alias) {
		return "", fmt.Errorf("%w: %q", ErrInvalidLibraryAlias, alias)
	}
	return alias, nil
}

// jsString renders s as a JavaScript string literal (JSON strings are valid JS).
func jsString(s string) string {
	b, _ := json.Marshal(s) // marshalling a string cannot fail
	return string(b)
}

// activateLibraryInSettings enables functionExternalModules (so function
// nodes can import the module from their Setup tab) and, when expose is true,
// adds `alias: require(module)` to functionGlobalContext. The require is
// guarded: a module that fails to load must not crash Node-RED at startup.
// Idempotent: a previous NRCC entry for the same module is replaced.
func activateLibraryInSettings(content, pkg, alias string, expose bool) string {
	content = replaceScalarKey(content, "functionExternalModules", "true")
	if !expose {
		return content
	}
	module := libraryModuleName(pkg)
	content = deactivateLibraryInSettings(content, module)
	entry := fmt.Sprintf("%s: (function () { try { return require(%s); } catch (err) { return undefined; } })(), %s%s",
		jsString(alias), jsString(module), libraryMarker, jsString(module))

	loc := globalContextOpenRe.FindStringIndex(content)
	if loc == nil {
		return appendPropertyBeforeClosingBrace(content, "  functionGlobalContext: {\n    "+entry+"\n  },\n")
	}
	rest := content[loc[1]:]
	if !strings.HasPrefix(rest, "\n") {
		// One-line block: keep the rest off the entry's trailing line comment.
		entry += "\n"
	}
	return content[:loc[1]] + "\n    " + entry + rest
}

// deactivateLibraryInSettings removes the NRCC-managed functionGlobalContext
// entry of module. functionExternalModules stays enabled: other libraries or
// operator flows may rely on it.
func deactivateLibraryInSettings(content, module string) string {
	re := regexp.MustCompile(`(?m)^[^\n]*` + regexp.QuoteMeta(libraryMarker+jsString(libraryModuleName(module))) + `[ \t]*\n`)
	return re.ReplaceAllLiteralString(content, "")
}

// isNodeRedNodePackage reports whether the installed package declares
// Node-RED nodes. Those load in the palette and are not exposed through
// functionGlobalContext. An unreadable package.json counts as a plain library.
func isNodeRedNodePackage(dir, module string) bool {
	// Callers pass a validated name; reject traversal here too so the read can
	// never leave node_modules.
	if strings.Contains(module, "..") {
		return false
	}
	// #nosec G304 -- module has no ".." (checked above); dir is the operator-supplied dataDir.
	data, err := os.ReadFile(filepath.Join(dir, "node_modules", module, "package.json"))
	if err != nil {
		return false
	}
	var manifest map[string]json.RawMessage
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	_, ok := manifest["node-red"]
	return ok
}
