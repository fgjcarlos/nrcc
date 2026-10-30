package main

// Default OCI label values.
const (
	defaultImageTitle       = "nrcc"
	defaultImageDescription = "Node-RED Control Center — all-in-one"
	defaultImageSource      = "https://github.com/fgjcarlos/nrcc"
)

// Target is one native build target of the release contract.
type Target struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	ARM      string `json:"arm,omitempty"`
	Platform string `json:"platform"`
	Binary   string `json:"binary"`
}

// Plan is the resolved release plan for one version.
type Plan struct {
	Version      string            `json:"version"`
	Tag          string            `json:"tag"`
	Channel      string            `json:"channel"`
	Prerelease   bool              `json:"prerelease"`
	Targets      []Target          `json:"targets"`
	Images       []string          `json:"images"`
	Tags         []string          `json:"tags"`
	MetadataTags []string          `json:"metadata_tags"`
	Labels       map[string]string `json:"labels"`
}

// supportedTargets returns the native build targets of the release
// contract.
//
// nrcc is Linux-only at compile time (ADR 0004): the backend uses
// POSIX-only syscall symbols, so a macOS or Windows binary cannot be
// produced. The two Linux architectures below are the ones the project
// supports and tests (CONTRIBUTING.md, "Supported platforms") and are
// exactly the ones the multi-architecture container image ships.
//
// linux/arm/v7 (32-bit ARM) is deliberately not published: the backend
// does not compile for 32-bit platforms, because syscall.Sysinfo_t
// exposes Totalram and Freeram as uint32 there while the metrics code
// assigns them to uint64 fields. Supporting 32-bit ARM means implementing
// and testing that platform, which is a separate piece of work; the
// release must not claim an artifact it cannot build.
func supportedTargets() []Target {
	return []Target{
		{OS: "linux", Arch: "amd64", Platform: "linux/amd64", Binary: "nrcc-linux-amd64"},
		{OS: "linux", Arch: "arm64", Platform: "linux/arm64", Binary: "nrcc-linux-arm64"},
	}
}

// isSupportedTarget reports whether a platform triple belongs to the
// release contract.
func isSupportedTarget(platform string) bool {
	for _, target := range supportedTargets() {
		if target.Platform == platform {
			return true
		}
	}
	return false
}

// newPlan resolves the release plan of a version.
func newPlan(version Version, images []string, revision, created string) Plan {
	plan := Plan{
		Version:    version.Semver,
		Tag:        version.Tag,
		Channel:    version.Channel(),
		Prerelease: version.IsPrerelease(),
		Targets:    supportedTargets(),
		Images:     images,
		Tags:       version.Tags(),
		Labels: map[string]string{
			"org.opencontainers.image.title":       defaultImageTitle,
			"org.opencontainers.image.description": defaultImageDescription,
			"org.opencontainers.image.source":      defaultImageSource,
			"org.opencontainers.image.version":     version.Semver,
		},
	}

	if revision != "" {
		plan.Labels["org.opencontainers.image.revision"] = revision
	}
	if created != "" {
		plan.Labels["org.opencontainers.image.created"] = created
	}

	for _, tag := range plan.Tags {
		plan.MetadataTags = append(plan.MetadataTags, "type=raw,value="+tag)
	}

	return plan
}
