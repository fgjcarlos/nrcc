package model

import (
	"fmt"
	"regexp"
)

// TenantID identifies a logical tenant. NRCC is mono-tenant by default; this
// type exists as a seam (ADR 0001) so multi-tenant work can attach later
// without branching tenant logic into every service today.
type TenantID string

// DefaultTenantID is the implicit tenant used by mono-tenant deployments. It
// resolves to the legacy flat DATA_DIR layout, so existing installs keep
// working unchanged. This value is part of the on-disk contract — do not change.
const DefaultTenantID TenantID = "default"

// TenantContext carries the resolved tenant for a request or operation. It is
// intentionally minimal for the first slice; richer fields can be added when
// tenant-aware behavior actually exists.
type TenantContext struct {
	ID TenantID
}

// tenantIDPattern restricts tenant IDs to a safe, single-path-segment charset:
// it must start with an alphanumeric and may contain only letters, digits,
// dashes and underscores (max 64 chars). This deliberately rejects path
// separators, "..", leading dots and control characters so a tenant ID can
// never escape its storage root — the path-traversal guard ADR 0001 requires.
var tenantIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// Validate reports whether the tenant ID is safe to use as a storage path
// segment. It returns an error describing why an ID is rejected.
func (id TenantID) Validate() error {
	if id == "" {
		return fmt.Errorf("tenant id must not be empty")
	}
	if !tenantIDPattern.MatchString(string(id)) {
		return fmt.Errorf("invalid tenant id %q: must match %s", string(id), tenantIDPattern.String())
	}
	return nil
}
