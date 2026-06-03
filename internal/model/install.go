package model

// InstallLayout defines the standard filesystem paths for nrcc installation
type InstallLayout struct {
	BinaryPath  string // /usr/local/bin/nrcc
	ConfigDir   string // /etc/nrcc
	EnvFile     string // /etc/nrcc/nrcc.env
	DataDir     string // /var/lib/nrcc
	SystemdUnit string // /etc/systemd/system/nrcc.service
	ServiceName string // nrcc
	ServiceUser string // nrcc
}

// NodeRedInstallMode controls what `nrcc install` should do before starting the service.
type NodeRedInstallMode string

const (
	NodeRedInstallModeSkip   NodeRedInstallMode = "skip"
	NodeRedInstallModeNative NodeRedInstallMode = "native"
	NodeRedInstallModeDocker NodeRedInstallMode = "docker"
)

// DefaultInstallLayout returns the standard production layout
func DefaultInstallLayout() InstallLayout {
	return InstallLayout{
		BinaryPath:  "/usr/local/bin/nrcc",
		ConfigDir:   "/etc/nrcc",
		EnvFile:     "/etc/nrcc/nrcc.env",
		DataDir:     "/var/lib/nrcc",
		SystemdUnit: "/etc/systemd/system/nrcc.service",
		ServiceName: "nrcc",
		ServiceUser: "nrcc",
	}
}

// InstallOpts provides options for the installation process.
//
// InstallOpts is the CLI/API input shape. It is intentionally mapped into an
// InstallPlan before side effects start so flag-driven installs and future TUI
// wizard installs share one execution path.
type InstallOpts struct {
	Layout             InstallLayout
	SkipPrompt         bool // for non-interactive/scripting use
	NodeRedMode        NodeRedInstallMode
	WithPortless       bool // install Portless CLI alongside nrcc
	PortlessQuickSetup bool // configure default Portless aliases after install
	PortlessTrust      bool // run Portless trust setup after install
}

// InstallPlan is the normalized, side-effect-free install decision model.
// Both the existing flag path and the guided wizard path produce this shape;
// InstallerService consumes only the plan after host detection and prompting.
type InstallPlan struct {
	SkipPrompt         bool
	NodeRedMode        NodeRedInstallMode
	NodeRedDetected    bool
	NodeRedCommand     string
	NodeRedUserDir     string
	NodeRedSettings    string
	WithPortless       bool
	PortlessQuickSetup bool
	PortlessTrust      bool
}

// NewInstallPlanFromOpts preserves the current flag-driven behavior while
// making it explicit and testable for future wizard front-ends.
func NewInstallPlanFromOpts(opts InstallOpts) InstallPlan {
	return InstallPlan{
		SkipPrompt:         opts.SkipPrompt,
		NodeRedMode:        opts.NodeRedMode,
		WithPortless:       opts.WithPortless,
		PortlessQuickSetup: opts.PortlessQuickSetup,
		PortlessTrust:      opts.PortlessTrust,
	}
}

// UninstallOpts provides options for the uninstall process
type UninstallOpts struct {
	Layout     InstallLayout
	KeepData   bool // --keep-data flag
	Purge      bool // --purge flag (remove data without prompting)
	SkipPrompt bool
}

// InstallStatus represents the current installation state
type InstallStatus struct {
	ServiceState   string // active, inactive, failed, not-installed, unknown
	DataDirExists  bool
	EnvFileExists  bool
	UnitFileExists bool
}
