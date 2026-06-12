package model

import "testing"

func TestDefaultInstanceID_IsStableConstant(t *testing.T) {
	// The synthesized default instance and the legacy single-instance layout both
	// depend on this exact ID; changing it would break instance selection.
	if DefaultInstanceID != "default" {
		t.Errorf("DefaultInstanceID = %q, want %q", DefaultInstanceID, "default")
	}
}

func TestInstanceKind_Valid(t *testing.T) {
	valid := []InstanceKind{InstanceKindLocal, InstanceKindDocker, InstanceKindSSH, InstanceKindAgent}
	for _, k := range valid {
		if !k.Valid() {
			t.Errorf("Valid(%q) = false, want true", k)
		}
	}
	invalid := []InstanceKind{"", "remote", "vm", "Local"}
	for _, k := range invalid {
		if k.Valid() {
			t.Errorf("Valid(%q) = true, want false", k)
		}
	}
}

func validInstance() Instance {
	return Instance{
		ID:     "default",
		Name:   "Default",
		Kind:   InstanceKindLocal,
		Health: InstanceHealthUnknown,
	}
}

func TestInstance_Validate_AcceptsValid(t *testing.T) {
	cases := []Instance{
		validInstance(),
		{ID: "prod-1", Name: "Prod", Kind: InstanceKindDocker, Health: InstanceHealthUnknown},
		{ID: "edge_2", Name: "Edge node", Kind: InstanceKindSSH, Health: InstanceHealthHealthy},
		{ID: "a.b", Name: "Dotted", Kind: InstanceKindAgent, Health: InstanceHealthUnhealthy},
	}
	for _, in := range cases {
		if err := in.Validate(); err != nil {
			t.Errorf("Validate(%+v) returned error, want nil: %v", in, err)
		}
	}
}

func TestInstance_Validate_RejectsBadID(t *testing.T) {
	// IDs become URL path segments, so they must be URL-safe and traversal-proof.
	badIDs := []string{"", "..", "../etc", "a/b", ".", "-lead", "has space", "a\x00b"}
	for _, id := range badIDs {
		in := validInstance()
		in.ID = id
		if err := in.Validate(); err == nil {
			t.Errorf("Validate with ID %q returned nil, want error", id)
		}
	}
}

func TestInstance_Validate_RejectsEmptyName(t *testing.T) {
	in := validInstance()
	in.Name = ""
	if err := in.Validate(); err == nil {
		t.Errorf("Validate with empty Name returned nil, want error")
	}
}

func TestInstance_Validate_RejectsUnknownKind(t *testing.T) {
	in := validInstance()
	in.Kind = "wormhole"
	if err := in.Validate(); err == nil {
		t.Errorf("Validate with unknown Kind returned nil, want error")
	}
}
