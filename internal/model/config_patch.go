package model

import (
	"encoding/json"
	"sort"
)

// ConfigPatch is a presence-aware partial configuration update.
//
// The structured apply endpoint accepts patches: the payload carries only
// the fields the operator actually changed. A field that is absent from the
// patch keeps its live value, a field present with an explicit null disables
// the feature it configures, and a field present with a value replaces the
// live value.
//
// The patch keeps the raw JSON of every field instead of a decoded struct
// because a decoded struct cannot express the difference between an omitted
// int and an int explicitly set to zero — the two carry different operator
// intents, and only the raw form keeps them apart.
type ConfigPatch struct {
	fields map[string]json.RawMessage
}

// NewConfigPatch builds a patch from the raw JSON fields of a request body.
// The fields are copied so a later mutation of the caller's map cannot
// change the patch.
func NewConfigPatch(fields map[string]json.RawMessage) ConfigPatch {
	copied := make(map[string]json.RawMessage, len(fields))
	for name, value := range fields {
		copied[name] = value
	}
	return ConfigPatch{fields: copied}
}

// Fields returns a copy of the raw patch fields.
func (p ConfigPatch) Fields() map[string]json.RawMessage {
	copied := make(map[string]json.RawMessage, len(p.fields))
	for name, value := range p.fields {
		copied[name] = value
	}
	return copied
}

// Keys returns the sorted names of the fields the patch carries.
func (p ConfigPatch) Keys() []string {
	names := make([]string, 0, len(p.fields))
	for name := range p.fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsEmpty reports whether the patch carries no field at all.
func (p ConfigPatch) IsEmpty() bool {
	return len(p.fields) == 0
}
