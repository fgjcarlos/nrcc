package service

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// Issue #880 — reconcile the structured state after a raw settings apply.
//
// Authoritative state contract:
//
//   - settings.js is authoritative for the values Node-RED reads. A raw
//     apply therefore defines the current configuration.
//   - config.json is the derived structured mirror of the values NRCC's
//     model represents, plus the NRCC-owned state the mirror holds:
//     envVars (with their encryption metadata), functionLibraries and
//     runtimeState.
//   - A successful raw apply re-derives the mirror from the applied bytes
//     inside the apply transaction, so a structured read reports what the
//     running Node-RED uses and the next visual save cannot revert the
//     raw change.
//
// The reconciliation never rewrites the source. Operator-owned blocks the
// structured model does not represent stay in settings.js, and the
// source-preserving patchers keep them verbatim.
//
// The source speaks for the values it declares: a key the source does not
// declare keeps its mirrored value instead of being reset to a parser
// default.

// reconcilableSourceKeys are the top-level settings.js keys the
// reconciliation reads back from the source. A key outside this list is
// operator-owned: it stays in settings.js and the mirror does not model it.
//
// `runtimeState` is deliberately absent: the structured model owns the
// runtime state and never renders it into the source, so the source never
// speaks for it.
var reconcilableSourceKeys = []string{
	"uiPort", "port", "uiHost", "httpAdminRoot", "httpNodeRoot",
	"flowFile", "userDir", "nodesDir", "lang", "disableEditor",
	"projectsEnabled", "credentialSecret", "requireHttps", "https",
	"adminAuth", "httpNodeAuth", "httpStaticAuth",
	"editorTheme", "env",
	// logging — issue #882. The source declares the logging values
	// Node-RED runs with, so the mirror adopts them like every other
	// managed block.
	"logging",
}

// sourceKeyPattern matches the top-level (and nested) keys a source
// declares. It is the fallback for a source the sandbox could not
// evaluate, where the exact top-level key set is unknown.
var sourceKeyPattern = regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\s*:`)

// ErrMirrorRejected reports that the authoritative source declares a
// configuration the structured mirror cannot hold. It is a validation
// failure, not an infrastructure failure: the source is rejected and the
// pre-write bytes are restored.
var ErrMirrorRejected = errors.New("settings.js: configuration cannot be mirrored")

// MirrorSnapshot captures the mirror state around a reconciliation so a
// failed transaction can restore it.
type MirrorSnapshot struct {
	// Existed reports whether the mirror file existed before the
	// reconciliation. A reconciliation that created the mirror is undone
	// by removing the file again.
	Existed bool
	// Previous is the mirror content before the reconciliation.
	Previous model.NodeRedConfig
	// Committed is the mirror content the reconciliation published.
	Committed model.NodeRedConfig
}

// ReconcileRawSettings re-derives the structured mirror from the
// authoritative settings.js content and commits it to the JSON store.
//
// The returned snapshot carries the previous and the committed mirror so a
// failed transaction can restore the previous state with RestoreMirror.
func (s *ConfigService) ReconcileRawSettings(content string) (MirrorSnapshot, error) {
	parsed, err := s.parseConfigFromContent(content)
	if err != nil {
		return MirrorSnapshot{}, err
	}
	exported, exportErr := exportSettingsObject(content)
	if exportErr != nil {
		// The source could not be evaluated. The reconciliation keeps the
		// mirrored value for the blocks only the sandbox can read and
		// falls back to the regex inspection for the declared keys.
		exported = nil
	}

	existed := s.store.Exists()
	var previous model.NodeRedConfig
	committed, err := s.Update(func(current *model.NodeRedConfig) error {
		previous = *current
		base := *current
		if !existed {
			// No mirror yet: the parsed source is the base the merge
			// starts from.
			base = parsed
		}
		candidate := reconcileConfigWithSource(base, parsed, content, exported)
		if err := s.Validate(candidate); err != nil {
			return fmt.Errorf("%w: %w", ErrMirrorRejected, err)
		}
		s.decorateConfig(&candidate)
		*current = candidate
		return nil
	})
	if err != nil {
		return MirrorSnapshot{}, err
	}
	return MirrorSnapshot{
		Existed:   existed,
		Previous:  previous,
		Committed: committed,
	}, nil
}

// RestoreMirror restores the mirror state captured by a reconciliation.
//
// A field is restored only when it still holds the reconciled value, so a
// concurrent update to another field survives the recovery.
func (s *ConfigService) RestoreMirror(snapshot MirrorSnapshot) error {
	if !snapshot.Existed {
		return s.store.Delete()
	}
	return s.store.Update(func(current *model.NodeRedConfig) error {
		live := reflect.ValueOf(current).Elem()
		previous := reflect.ValueOf(snapshot.Previous)
		committed := reflect.ValueOf(snapshot.Committed)
		for i := 0; i < live.NumField(); i++ {
			name := live.Type().Field(i).Name
			switch name {
			case "FunctionLibraries", "RuntimeState", "SettingsPath", "SettingsSource", "Extra":
				continue
			}
			if reflect.DeepEqual(live.Field(i).Interface(), committed.Field(i).Interface()) {
				live.Field(i).Set(previous.Field(i))
			}
		}
		return nil
	})
}

// reconcileConfigWithSource merges the values the source declares into the
// current mirror state.
//
// The source wins for every managed key it declares. A key the source does
// not declare keeps its mirrored value. The NRCC-owned state the mirror
// holds (environment metadata, function libraries, runtime state) is never
// taken from the source.
func reconcileConfigWithSource(current, parsed model.NodeRedConfig, content string, exported map[string]any) model.NodeRedConfig {
	declared := declaredSourceKeys(content, exported)
	merged := current

	if declared["uiPort"] || declared["port"] {
		merged.Port = parsed.Port
		merged.UIPort = parsed.UIPort
	}
	if declared["uiHost"] {
		merged.UIHost = parsed.UIHost
	}
	if declared["httpAdminRoot"] {
		merged.HTTPAdminRoot = parsed.HTTPAdminRoot
	}
	if declared["httpNodeRoot"] {
		merged.HTTPNodeRoot = parsed.HTTPNodeRoot
	}
	if declared["flowFile"] {
		merged.FlowFile = parsed.FlowFile
	}
	if declared["userDir"] {
		merged.UserDir = parsed.UserDir
	}
	if declared["nodesDir"] {
		merged.NodesDir = parsed.NodesDir
	}
	if declared["lang"] {
		merged.Lang = parsed.Lang
	}
	if declared["disableEditor"] {
		merged.DisableEditor = parsed.DisableEditor
	}
	if declared["projectsEnabled"] {
		merged.ProjectsEnabled = parsed.ProjectsEnabled
	}
	if declared["credentialSecret"] {
		merged.CredentialSecret = parsed.CredentialSecret
	}
	if declared["requireHttps"] {
		merged.RequireHttps = parsed.RequireHttps
	}
	if declared["https"] {
		merged.Https = parsed.Https
	}
	if declared["logging"] {
		merged.Logging = parsed.Logging
	}
	if declared["adminAuth"] {
		merged.AdminAuth = parsed.AdminAuth
	}
	if declared["httpNodeAuth"] {
		merged.HTTPNodeAuth = parsed.HTTPNodeAuth
	}
	if declared["httpStaticAuth"] {
		merged.HTTPStaticAuth = parsed.HTTPStaticAuth
	}

	if exported != nil {
		if declared["editorTheme"] {
			if value, ok := exported["editorTheme"]; ok {
				merged.EditorTheme = value
			}
		}
		if declared["env"] {
			if value, ok := exported["env"]; ok {
				merged.EnvVars = mergeEnvVars(current.EnvVars, envVarsFromSource(value))
			}
		}
	}

	// The runtime state, the function libraries and the derived settings
	// coordinates are owned by NRCC: the model never renders them into the
	// source, so the source never speaks for them.
	merged.RuntimeState = current.RuntimeState
	merged.FunctionLibraries = current.FunctionLibraries
	merged.SettingsPath = current.SettingsPath
	merged.SettingsSource = current.SettingsSource
	merged.Extra = current.Extra
	return merged
}

// declaredSourceKeys reports which managed keys the source declares.
//
// The sandbox export is authoritative when the source could be evaluated: it
// knows the exact top-level keys. The regex scan is the fallback for a
// source the sandbox could not evaluate.
func declaredSourceKeys(content string, exported map[string]any) map[string]bool {
	declared := make(map[string]bool, len(reconcilableSourceKeys))
	if exported != nil {
		for _, key := range reconcilableSourceKeys {
			if _, ok := exported[key]; ok {
				declared[key] = true
			}
		}
		return declared
	}
	for _, match := range sourceKeyPattern.FindAllStringSubmatch(content, -1) {
		declared[match[1]] = true
	}
	return declared
}

// mergeEnvVars adopts the values the source declares for the environment
// keys NRCC already manages and keeps the NRCC-owned metadata of every key:
// the type, the description, the encryption state and the origin.
//
// A key the source does not declare keeps its mirrored value: the mirror is
// the only place the NRCC-owned environment metadata lives.
//
// A key NRCC stores encrypted keeps NRCC's stored value. The source never
// renders an encrypted value, so a plaintext there is not NRCC's to adopt.
func mergeEnvVars(current, source []model.EnvVar) []model.EnvVar {
	merged := make([]model.EnvVar, 0, len(current)+len(source))
	index := make(map[string]int, len(current))
	for _, envVar := range current {
		index[envVar.Key] = len(merged)
		merged = append(merged, envVar)
	}
	for _, envVar := range source {
		position, known := index[envVar.Key]
		if !known {
			index[envVar.Key] = len(merged)
			merged = append(merged, envVar)
			continue
		}
		existing := merged[position]
		if existing.Encrypted || existing.Type == "secret" {
			continue
		}
		existing.Value = envVar.Value
		if envVar.Type != "" {
			existing.Type = envVar.Type
		}
		merged[position] = existing
	}
	return merged
}

// envVarsFromSource converts an exported env block into environment
// variables. Node-RED accepts an object map and NRCC renders an array of
// {name, value, type} entries; both shapes are supported.
func envVarsFromSource(value any) []model.EnvVar {
	switch block := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(block))
		for key := range block {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		vars := make([]model.EnvVar, 0, len(keys))
		for _, key := range keys {
			vars = append(vars, model.EnvVar{
				Key:   key,
				Value: envValueString(block[key]),
				Type:  envValueType(block[key]),
			})
		}
		return vars
	case []any:
		vars := make([]model.EnvVar, 0, len(block))
		for _, entry := range block {
			fields, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := fields["name"].(string)
			if name == "" {
				continue
			}
			envVar := model.EnvVar{Key: name}
			if text, ok := fields["value"].(string); ok {
				envVar.Value = text
			} else {
				envVar.Value = envValueString(fields["value"])
			}
			if kind, ok := fields["type"].(string); ok {
				envVar.Type = envTypeFromSource(kind)
			} else {
				envVar.Type = envValueType(fields["value"])
			}
			vars = append(vars, envVar)
		}
		return vars
	default:
		return nil
	}
}

// envValueString renders an exported environment value as text.
func envValueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

// envValueType maps an exported environment value onto the NRCC type
// vocabulary: "string" | "number" | "boolean" | "secret".
func envValueType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	default:
		return "string"
	}
}

// envTypeFromSource maps a type declared by the source onto the NRCC type
// vocabulary. The source may use the Node-RED spellings.
func envTypeFromSource(kind string) string {
	switch kind {
	case "str", "string":
		return "string"
	case "num", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "secret":
		return "secret"
	default:
		return "string"
	}
}
