package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Publication channels of the release contract (issue #884).
const (
	// ChannelStable publishes the floating "latest" tag.
	ChannelStable = "stable"
	// ChannelPrerelease publishes the floating "candidate" tag and never
	// touches "latest".
	ChannelPrerelease = "prerelease"
)

// Floating Docker tags of the release contract.
const (
	// TagLatest is the stable channel tag.
	TagLatest = "latest"
	// TagCandidate is the prerelease channel tag.
	TagCandidate = "candidate"
)

// Version is a parsed release version.
//
// The numeric components are integers so the struct stays small enough to
// be passed by value; the canonical string form is kept in Semver.
type Version struct {
	// Tag is the release tag as provided, for example "v1.2.3-rc.1".
	Tag string
	// Semver is the normalised version, for example "1.2.3-rc.1".
	Semver string
	// Prerelease holds the prerelease suffix without the leading dash.
	Prerelease string
	Major      int
	Minor      int
	Patch      int
}

// IsPrerelease reports whether the version carries a prerelease suffix.
func (v Version) IsPrerelease() bool {
	return v.Prerelease != ""
}

// Channel returns the publication channel of the version.
func (v Version) Channel() string {
	if v.IsPrerelease() {
		return ChannelPrerelease
	}
	return ChannelStable
}

// Tags returns the Docker tags published for the version.
//
// A stable version publishes its own version, the floating major.minor tag
// and "latest". A prerelease publishes its own version and "candidate" only:
// it never takes over "latest" and never moves the floating major.minor tag.
func (v Version) Tags() []string {
	if v.IsPrerelease() {
		return []string{v.Semver, TagCandidate}
	}
	return []string{
		v.Semver,
		fmt.Sprintf("%d.%d", v.Major, v.Minor),
		TagLatest,
	}
}

// parseVersion parses a release version such as "v1.2.3" or "1.2.3-rc.1".
//
// Build metadata ("+meta") is accepted and dropped: it is not part of the
// published version. Everything else that is not a strict
// MAJOR.MINOR.PATCH[-prerelease] string is rejected so a malformed tag can
// never reach a registry.
func parseVersion(raw string) (Version, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Version{}, errors.New("version is empty")
	}

	tag := trimmed
	body := strings.TrimPrefix(strings.TrimPrefix(trimmed, "v"), "V")
	if body == "" {
		return Version{}, fmt.Errorf("version %q has no numeric part", raw)
	}

	if idx := strings.Index(body, "+"); idx >= 0 {
		body = body[:idx]
	}

	core := body
	prerelease := ""
	if idx := strings.Index(body, "-"); idx >= 0 {
		core = body[:idx]
		prerelease = body[idx+1:]
		if prerelease == "" {
			return Version{}, fmt.Errorf("version %q has an empty prerelease suffix", raw)
		}
		if err := validatePrerelease(prerelease); err != nil {
			return Version{}, fmt.Errorf("version %q: %w", raw, err)
		}
	}

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("version %q is not a MAJOR.MINOR.PATCH version", raw)
	}

	numbers := make([]int, 3)
	for i, part := range parts {
		number, err := parseNumericComponent(part)
		if err != nil {
			return Version{}, fmt.Errorf("version %q: %w", raw, err)
		}
		numbers[i] = number
	}

	semver := fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2])
	if prerelease != "" {
		semver += "-" + prerelease
	}

	return Version{
		Tag:        tag,
		Semver:     semver,
		Prerelease: prerelease,
		Major:      numbers[0],
		Minor:      numbers[1],
		Patch:      numbers[2],
	}, nil
}

// parseNumericComponent parses one numeric version component and rejects
// anything that is not a canonical decimal number.
func parseNumericComponent(part string) (int, error) {
	if part == "" {
		return 0, errors.New("version has an empty numeric component")
	}
	if len(part) > 1 && part[0] == '0' {
		return 0, fmt.Errorf("version component %q has a leading zero", part)
	}
	for _, r := range part {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("version component %q is not numeric", part)
		}
	}
	number, err := strconv.Atoi(part)
	if err != nil {
		return 0, fmt.Errorf("version component %q is not numeric", part)
	}
	return number, nil
}

// validatePrerelease rejects prerelease suffixes that a container registry
// would not accept as a tag.
func validatePrerelease(prerelease string) error {
	for _, r := range prerelease {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '.' || r == '-' || r == '_':
		default:
			return fmt.Errorf("prerelease suffix %q contains an unsupported character %q", prerelease, string(r))
		}
	}
	return nil
}
