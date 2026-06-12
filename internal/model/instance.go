package model

import (
	"fmt"
	"regexp"
	"time"
)

// InstanceKind identifies the transport/control model of a managed Node-RED
// instance. Only "local" is operational today; the others are reserved by the
// multi-instance architecture (docs/architecture/multi-instance-node-red.md)
// so the model is forward-compatible without enabling those behaviors yet.
type InstanceKind string

const (
	InstanceKindLocal  InstanceKind = "local"
	InstanceKindDocker InstanceKind = "docker"
	InstanceKindSSH    InstanceKind = "ssh"
	InstanceKindAgent  InstanceKind = "agent"
)

// Valid reports whether the kind is one of the known instance kinds.
func (k InstanceKind) Valid() bool {
	switch k {
	case InstanceKindLocal, InstanceKindDocker, InstanceKindSSH, InstanceKindAgent:
		return true
	default:
		return false
	}
}

// Instance health states. Real probing is wired in a later slice; the first
// read-only slice reports Unknown for the synthesized default.
const (
	InstanceHealthUnknown   = "unknown"
	InstanceHealthHealthy   = "healthy"
	InstanceHealthUnhealthy = "unhealthy"
)

// DefaultInstanceID is the stable ID of the implicit single-instance default,
// which maps to the existing DATA_DIR. Existing installs always see this one.
const DefaultInstanceID = "default"

// Instance describes a managed Node-RED control boundary. See
// docs/architecture/multi-instance-node-red.md (#144).
type Instance struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Kind        InstanceKind `json:"kind"`
	DataDir     string       `json:"dataDir,omitempty"`
	BaseURL     string       `json:"baseUrl,omitempty"`
	Health      string       `json:"health"`
	AuthContext string       `json:"authContext,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

// instanceIDPattern restricts instance IDs to a URL-safe, single-path-segment
// charset: it must start with an alphanumeric and may contain letters, digits,
// dashes, underscores and dots. This rejects path separators, "..", leading
// dots/dashes and control characters so an ID can never escape an allowed root
// when it later becomes a `/api/instances/{id}` path segment.
var instanceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// Validate reports whether the instance is well-formed: URL-safe ID, non-empty
// name, and a known kind.
func (i Instance) Validate() error {
	if !instanceIDPattern.MatchString(i.ID) {
		return fmt.Errorf("invalid instance id %q: must match %s", i.ID, instanceIDPattern.String())
	}
	if i.Name == "" {
		return fmt.Errorf("instance name must not be empty")
	}
	if !i.Kind.Valid() {
		return fmt.Errorf("invalid instance kind %q", i.Kind)
	}
	return nil
}
