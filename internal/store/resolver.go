package store

import (
	"path/filepath"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TenantPathResolver maps a TenantID to its on-disk storage root.
//
// It is the store-boundary seam from ADR 0001. For the default tenant it
// resolves to the legacy flat root (the current DATA_DIR layout), guaranteeing
// zero behavior change for existing mono-tenant deployments. Named tenants
// resolve under <root>/tenants/<id>, which only matters once multi-tenant
// persistence is actually enabled.
type TenantPathResolver struct {
	root string
}

// NewTenantPathResolver creates a resolver rooted at the given data directory
// (typically the value of DATA_DIR).
func NewTenantPathResolver(root string) *TenantPathResolver {
	return &TenantPathResolver{root: root}
}

// Resolve returns the storage root directory for the given tenant.
//
// The default tenant resolves to the legacy root unchanged; named tenants
// resolve to <root>/tenants/<id>. Invalid tenant IDs are rejected (returning an
// empty path and an error) to prevent path traversal out of the data root.
func (r *TenantPathResolver) Resolve(id model.TenantID) (string, error) {
	if err := id.Validate(); err != nil {
		return "", err
	}
	if id == model.DefaultTenantID {
		return r.root, nil
	}
	return filepath.Join(r.root, "tenants", string(id)), nil
}
