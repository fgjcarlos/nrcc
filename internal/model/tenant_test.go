package model

import "testing"

func TestDefaultTenantID_IsStableConstant(t *testing.T) {
	// The legacy mono-tenant layout depends on this exact value: any change here
	// would silently relocate every existing deployment's data root.
	if DefaultTenantID != "default" {
		t.Errorf("DefaultTenantID = %q, want %q", DefaultTenantID, "default")
	}
}

func TestDefaultTenantID_Validates(t *testing.T) {
	if err := DefaultTenantID.Validate(); err != nil {
		t.Errorf("DefaultTenantID must be a valid tenant id, got error: %v", err)
	}
}

func TestTenantContext_CarriesID(t *testing.T) {
	ctx := TenantContext{ID: DefaultTenantID}
	if ctx.ID != DefaultTenantID {
		t.Errorf("TenantContext.ID = %q, want %q", ctx.ID, DefaultTenantID)
	}
}

func TestTenantID_Validate_AcceptsSafeIDs(t *testing.T) {
	valid := []TenantID{
		"default",
		"acme",
		"tenant-1",
		"tenant_2",
		"AB12",
		"a",
	}
	for _, id := range valid {
		if err := id.Validate(); err != nil {
			t.Errorf("Validate(%q) returned error, want nil: %v", id, err)
		}
	}
}

func TestTenantID_Validate_RejectsUnsafeIDs(t *testing.T) {
	// Anything that could escape the storage root or break a path segment must
	// be rejected — this is the path-traversal guard ADR 0001 calls out.
	invalid := []TenantID{
		"",           // empty
		"..",         // parent dir
		"../etc",     // traversal
		"a/b",        // separator
		"a\\b",       // windows separator
		".",          // current dir
		".hidden",    // leading dot
		"-leading",   // leading dash (not alnum)
		"with space", // whitespace
		"a\x00b",     // null byte
		"t\tab",      // control char
	}
	for _, id := range invalid {
		if err := id.Validate(); err == nil {
			t.Errorf("Validate(%q) returned nil, want error", id)
		}
	}
}
