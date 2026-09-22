package service

import (
	"errors"
	"fmt"
	"sort"
)

// Surface names a Node-RED surface that a preset is allowed to touch.
// The set is intentionally closed: a preset that declares no surface
// (or an unknown surface) is rejected by the surface-contract test
// (see TestPresetSurfaceContract).
//
// Surface boundaries are the whole point of issue #764's preset
// contract: a preset that touches the editor must never silently
// rewrite an HTTP-only setting, and vice versa.
type Surface string

const (
	// SurfaceEditor covers the Node-RED editor surface (admin UI
	// served by httpAdminRoot) — palette, projects, deploy button,
	// editorTheme, logging handlers, etc.
	SurfaceEditor Surface = "editor"
	// SurfaceHTTP covers the runtime HTTP API (httpNodeRoot +
	// httpNodeAuth + httpStatic + https + functionGlobalContext +
	// adminAuth middleware). A preset that touches httpNodeAuth
	// must declare this surface.
	SurfaceHTTP Surface = "http"
	// SurfaceSocketIO covers the runtime Socket.IO transport used by
	// the editor and nodes for live updates. Node-RED 5 exposes a
	// narrow surface (mostly ioPlugin opts); slice 1 declares the
	// constant for forward compatibility but does not yet ship a
	// socketio-only preset.
	SurfaceSocketIO Surface = "socketio"
)

// TrustTier reports whether a preset is operator-managed (writes only
// through the managed-key contract enforced by source_patch.go) or
// curated (an operator-vetted recipe that touches one or more managed
// keys under a documented trust boundary).
type TrustTier string

const (
	// TrustManaged presets produce edits that go through SourcePatch
	// and the apply pipeline with no extra trust boundary beyond
	// the existing admin role gate. This is the default for slice 1.
	TrustManaged TrustTier = "managed"
	// TrustCurated presets carry an additional "operator-vetted"
	// marker (visible in the UI as a "verified" badge). The marker
	// does not change the apply pipeline; it exists so the UI can
	// distinguish "preset we ship" from "preset an admin authored
	// through the registry's Register API". Slice 1 ships all four
	// presets as TrustManaged; TrustCurated is declared for future
	// use so the contract is locked in.
	TrustCurated TrustTier = "curated"
)

// Preset describes a curated settings.js patch recipe.
//
// The contract (enforced by TestPresetSurfaceContract and the per-preset
// tests in preset_registry_test.go):
//
//   - ID is unique within a registry.
//   - ManagedKeys is a non-empty subset of ManagedSettingKeys(); the
//     preset never writes a key outside that set.
//   - Surfaces is a non-empty subset of {SurfaceEditor, SurfaceHTTP,
//     SurfaceSocketIO}; each ManagedKey must align with at least one
//     declared surface (slice 2 will harden this with a per-key
//     surface map; slice 1 accepts surface declarations as a coarse
//     signal).
//   - BuildEdits is pure: given identical currentValues it returns
//     identical SourceEdits. Slice 2 will thread the live document
//     through BuildEdits for preview; slice 1's BuildEdits returns a
//     fixed placeholder edit sufficient to round-trip through the
//     apply pipeline.
type Preset struct {
	ID              string
	Name            string
	Description     string
	Surfaces        []Surface
	Trust           TrustTier
	ChannelBoundary string
	ManagedKeys     []string
	BuildEdits      func(content string, currentValues map[string]string) ([]SourceEdit, error)
}

// PresetRegistry is a defensive lookup for curated presets.
//
// The zero value is NOT ready; use NewPresetRegistry. Duplicate IDs
// fail loudly via Register so a stray "import two preset packages"
// can't silently overwrite a preset the operator trusts.
type PresetRegistry struct {
	presets map[string]Preset
}

// NewPresetRegistry returns a registry populated with the four core
// presets shipped with slice 1 of issue #764. The set is intentionally
// small; the issue's outcome is "vetted high-value recipes", not
// "every callback under the sun". Slice 2 may add more; slice 3 will
// surface them in the UI.
func NewPresetRegistry() *PresetRegistry {
	r := &PresetRegistry{presets: map[string]Preset{}}
	for _, p := range corePresets() {
		// corePresets() returns pre-validated, non-duplicate IDs,
		// so the error is unreachable. We swallow it explicitly
		// rather than panicking so a future refactor that splits
		// corePresets() across files cannot crash a production
		// startup on a duplicate.
		_ = r.Register(p)
	}
	return r
}

// Register adds p to the registry. Returns an error if p.ID is empty
// or already registered. Callers that want a "best effort" registration
// can ignore the error; callers that want strict ordering should fail
// fast.
func (r *PresetRegistry) Register(p Preset) error {
	if p.ID == "" {
		return errors.New("preset ID must not be empty")
	}
	if _, exists := r.presets[p.ID]; exists {
		return fmt.Errorf("preset %q already registered", p.ID)
	}
	r.presets[p.ID] = p
	return nil
}

// Get returns the preset registered under id. The boolean reports
// presence; the value is the zero Preset when false. Registry lookups
// never panic on a missing id.
func (r *PresetRegistry) Get(id string) (Preset, bool) {
	p, ok := r.presets[id]
	return p, ok
}

// List returns all registered presets sorted by ID so the UI can
// render a deterministic menu without resorting client-side. The
// returned slice is a fresh copy; callers may mutate it freely.
func (r *PresetRegistry) List() []Preset {
	out := make([]Preset, 0, len(r.presets))
	for _, p := range r.presets {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// corePresets returns the curated presets shipped with slice 1. The
// order is informational — NewPresetRegistry sorts via List — but kept
// here in a readable order so a code reviewer sees "https, then FGC
// strict, then FGC relaxed, then logging".
func corePresets() []Preset {
	return []Preset{
		httpsTLSPreset(),
		functionGlobalContextStrictPreset(),
		functionGlobalContextRelaxedPreset(),
		loggingCallbackMiddlewarePreset(),
	}
}

// httpsTLSPreset manages the top-level https block (key, cert,
// optional passphrase, optional ca). Slice 1 ships a placeholder block
// so the apply pipeline can round-trip it; slice 2 will derive the
// block from currentValues plus operator-supplied inputs.
func httpsTLSPreset() Preset {
	return Preset{
		ID:              "https-tls-preset",
		Name:            "HTTPS / TLS configuration",
		Description:     "Manage the top-level https block (key, cert, optional passphrase and CA). Touches the HTTP surface only.",
		Surfaces:        []Surface{SurfaceHTTP},
		Trust:           TrustManaged,
		ChannelBoundary: "http-only",
		ManagedKeys:     []string{"https"},
		BuildEdits: func(content string, currentValues map[string]string) ([]SourceEdit, error) {
			// Slice 1: emit a fixed placeholder block. Slice 2 will
			// derive the block content from currentValues plus the
			// operator's inputs (key path, cert path, etc.) and the
			// readiness probe for the live document.
			_ = content
			_ = currentValues
			return []SourceEdit{{
				Key:     "https",
				IsBlock: true,
				Block: "https: {\n" +
					"  key: $KEY_PATH,\n" +
					"  cert: $CERT_PATH,\n" +
					"}",
			}}, nil
		},
	}
}

// functionGlobalContextStrictPreset manages functionGlobalContext with
// a strict allow-list of safe keys. Operators who picked
// functionGlobalContext-loose on a prior install keep it; this preset
// is the safe default going forward.
func functionGlobalContextStrictPreset() Preset {
	return Preset{
		ID:              "functionGlobalContext-strict",
		Name:            "functionGlobalContext (strict)",
		Description:     "Manage functionGlobalContext with a curated allow-list of safe keys. Touches the editor surface (palette uses FGC).",
		Surfaces:        []Surface{SurfaceEditor},
		Trust:           TrustManaged,
		ChannelBoundary: "editor-only",
		ManagedKeys:     []string{"functionGlobalContext"},
		BuildEdits: func(content string, currentValues map[string]string) ([]SourceEdit, error) {
			// The strict preset intentionally REPLACES the FGC block
			// with a curated allow-list. Unknown / executable keys
			// (e.g. require() expressions) are not preserved — that
			// is the whole point of "strict". The relaxed sibling
			// preserves them.
			_ = content
			_ = currentValues
			return []SourceEdit{{
				Key:     "functionGlobalContext",
				IsBlock: true,
				Block: "functionGlobalContext: {\n" +
					"  // curated keys only\n" +
					"}",
			}}, nil
		},
	}
}

// functionGlobalContextRelaxedPreset is the permissive sibling of
// FGC-strict. It allows arbitrary keys but still routes through the
// managed-key contract so unmanaged code outside functionGlobalContext
// stays byte-stable.
func functionGlobalContextRelaxedPreset() Preset {
	return Preset{
		ID:              "functionGlobalContext-relaxed",
		Name:            "functionGlobalContext (relaxed)",
		Description:     "Manage functionGlobalContext with arbitrary keys permitted. Existing custom code outside the managed key stays byte-stable.",
		Surfaces:        []Surface{SurfaceEditor},
		Trust:           TrustManaged,
		ChannelBoundary: "editor-only",
		ManagedKeys:     []string{"functionGlobalContext"},
		BuildEdits: func(content string, currentValues map[string]string) ([]SourceEdit, error) {
			// Slice 2: when the existing FGC block is a recognised
			// module.exports top-level object literal, round-trip the
			// operator-authored entries (e.g. require() expressions)
			// through the apply so the issue acceptance test
			// (FunctionGlobalContextAndNodeDefaultsFixtureSuite)
			// passes the "executable/unknown cases remain preserved"
			// half. When the block is absent or unparseable, fall
			// back to a minimal skeleton.
			existing := extractTopLevelBlockContent(content, "functionGlobalContext")
			if existing == "" {
				return []SourceEdit{{
					Key:     "functionGlobalContext",
					IsBlock: true,
					Block: "functionGlobalContext: {\n" +
						"  // operator-managed keys\n" +
						"}",
				}}, nil
			}
			_ = currentValues
			return []SourceEdit{{
				Key:     "functionGlobalContext",
				IsBlock: true,
				Block:   existing,
			}}, nil
		},
	}
}

// loggingCallbackMiddlewarePreset manages the logging block, including
// the supported custom log handler callback shape. Touches the editor
// surface (log handlers appear in the editor log panel).
func loggingCallbackMiddlewarePreset() Preset {
	return Preset{
		ID:              "logging-callback-middleware",
		Name:            "Logging callbacks and middleware",
		Description:     "Manage the logging block with custom handlers and audit middleware. Touches the editor surface.",
		Surfaces:        []Surface{SurfaceEditor},
		Trust:           TrustManaged,
		ChannelBoundary: "editor-only",
		ManagedKeys:     []string{"logging"},
		BuildEdits: func(content string, currentValues map[string]string) ([]SourceEdit, error) {
			_ = content
			_ = currentValues
			return []SourceEdit{{
				Key:     "logging",
				IsBlock: true,
				Block: "logging: {\n" +
					"  console: {\n" +
					"    level: 'info',\n" +
					"  },\n" +
					"}",
			}}, nil
		},
	}
}
