package service

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// MergePatch applies a presence-aware patch on top of the live configuration
// and returns the merged result.
//
// The merge walks the JSON representation of both sides so the three
// operator intents stay distinguishable (issue #879):
//
//   - a field absent from the patch keeps its live value;
//   - a field present with a JSON object merges key by key, so a patch that
//     only replaces the `users` of `adminAuth` keeps the auth type and the
//     session expiry;
//   - a field present with any other value — `null` included — replaces the
//     live value, so `adminAuth: null` disables the admin surface.
//
// Arrays always replace: the `users` list of `adminAuth` is owned by the
// patch and is never merged element by element.
//
// The merge is pure: it reads the base configuration and the patch and
// returns a new configuration without touching the store.
func MergePatch(base model.NodeRedConfig, patch model.ConfigPatch) (model.NodeRedConfig, error) {
	if patch.IsEmpty() {
		return base, nil
	}
	baseFields, err := configJSONFields(base)
	if err != nil {
		return model.NodeRedConfig{}, fmt.Errorf("encode live configuration: %w", err)
	}
	patchFields := patch.Fields()
	for _, name := range patch.Keys() {
		baseFields[name] = mergeJSONValue(baseFields[name], patchFields[name])
	}
	// NRCC's typed configuration keeps `port` and `uiPort` in sync: the
	// settings.js the renderer writes carries the editor port as `uiPort`,
	// and the typed model derives `port` from it. A patch that moves
	// `uiPort` therefore moves `port` with it, exactly like a payload
	// that carries only `uiPort` does. A patch that carries `port`
	// explicitly keeps both independent.
	if _, explicit := patchFields["port"]; !explicit {
		if uiPort, ok := patchFields["uiPort"]; ok {
			baseFields["port"] = uiPort
		}
	}
	merged, err := json.Marshal(baseFields)
	if err != nil {
		return model.NodeRedConfig{}, fmt.Errorf("encode merged configuration: %w", err)
	}
	var out model.NodeRedConfig
	if err := json.Unmarshal(merged, &out); err != nil {
		return model.NodeRedConfig{}, fmt.Errorf("decode merged configuration: %w", err)
	}
	return out, nil
}

// configJSONFields renders the configuration as a map of raw JSON fields so
// the merge can overlay single fields without losing the ones the patch
// does not mention.
func configJSONFields(cfg model.NodeRedConfig) (map[string]json.RawMessage, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// mergeJSONValue merges one patch value onto the live value. Two JSON
// objects merge key by key; every other combination replaces the live value,
// which is what makes an explicit null an opt-out.
func mergeJSONValue(live, patch json.RawMessage) json.RawMessage {
	if !isJSONObject(live) || !isJSONObject(patch) {
		return patch
	}
	liveFields := map[string]json.RawMessage{}
	if err := json.Unmarshal(live, &liveFields); err != nil {
		return patch
	}
	patchFields := map[string]json.RawMessage{}
	if err := json.Unmarshal(patch, &patchFields); err != nil {
		return patch
	}
	for name, value := range patchFields {
		liveFields[name] = mergeJSONValue(liveFields[name], value)
	}
	merged, err := json.Marshal(liveFields)
	if err != nil {
		return patch
	}
	return merged
}

// isJSONObject reports whether the raw value is a JSON object.
func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
