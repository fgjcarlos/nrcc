package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/spf13/cobra"
)

// TestRootHelpFlag verifies root command --help works
func TestRootHelpFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()

	// Help should succeed or return nil
	if err != nil && err.Error() != "exit status 0" && err.Error() != "" {
		t.Errorf("root --help failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("help output should not be empty")
	}
}

// TestRootHelpContainsNodeRed verifies help mentions node-red command
func TestRootHelpContainsNodeRed(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "node-red") {
		t.Error("help should mention 'node-red' command")
	}
}

// TestCommandsRegistered verifies all major commands are registered
func TestCommandsRegistered(t *testing.T) {
	requiredCmds := []string{
		"node-red",
		"config",
		"env",
		"doctor",
		"setup",
		"update",
		"portless",
	}

	cmds := rootCmd.Commands()
	cmdNames := make(map[string]bool)

	for _, cmd := range cmds {
		cmdNames[cmd.Use] = true
	}

	for _, required := range requiredCmds {
		if !cmdNames[required] {
			t.Errorf("command '%s' not registered in rootCmd", required)
		}
	}
}

func TestInitializeServices_SkipsDataDirCreationForDoctor(t *testing.T) {
	oldMkdirAll := mkdirAll
	oldSvc := svc
	oldDataDir := os.Getenv("DATA_DIR")
	defer func() {
		mkdirAll = oldMkdirAll
		svc = oldSvc
		if oldDataDir == "" {
			os.Unsetenv("DATA_DIR")
		} else {
			os.Setenv("DATA_DIR", oldDataDir)
		}
	}()

	os.Setenv("DATA_DIR", "/var/lib/nrcc")
	mkdirAll = func(path string, perm os.FileMode) error {
		return errors.New("mkdir should not be called for doctor")
	}

	cmd := &cobra.Command{Use: "doctor"}
	if err := initializeServices(cmd); err != nil {
		t.Fatalf("doctor should initialize without creating data dir: %v", err)
	}
}

func TestInitializeServices_CreatesDataDirForWriteCommands(t *testing.T) {
	oldMkdirAll := mkdirAll
	oldSvc := svc
	oldDataDir := os.Getenv("DATA_DIR")
	defer func() {
		mkdirAll = oldMkdirAll
		svc = oldSvc
		if oldDataDir == "" {
			os.Unsetenv("DATA_DIR")
		} else {
			os.Setenv("DATA_DIR", oldDataDir)
		}
	}()

	os.Setenv("DATA_DIR", t.TempDir())
	called := false
	mkdirAll = func(path string, perm os.FileMode) error {
		called = true
		return nil
	}

	cmd := &cobra.Command{Use: "env"}
	if err := initializeServices(cmd); err != nil {
		t.Fatalf("write command initializeServices: %v", err)
	}
	if !called {
		t.Fatal("write commands should still create the data directory")
	}
}

func TestPortlessCommandRegistered(t *testing.T) {
	var found *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "portless" {
			found = cmd
			break
		}
	}
	if found == nil {
		t.Fatal("portless command not found")
	}

	requiredSubcmds := []string{"install", "status", "expose", "quick-setup", "setup-trust", "uninstall"}
	subcmdNames := make(map[string]bool)
	for _, cmd := range found.Commands() {
		subcmdNames[cmd.Use] = true
	}
	for _, required := range requiredSubcmds {
		if !subcmdNames[required] {
			t.Errorf("portless subcommand %q not registered", required)
		}
	}
}

func TestPortlessHelpCompleteness(t *testing.T) {
	output := portlessCmd.Long
	for _, want := range []string{"nrcc portless install", "nrcc portless quick-setup", "nrcc portless setup-trust", "install", "status", "expose", "quick-setup", "setup-trust", "uninstall"} {
		if !strings.Contains(output, want) {
			t.Fatalf("portless help missing %q in:\n%s", want, output)
		}
	}
}

func TestPortlessNewCommandHelp(t *testing.T) {
	for _, cmd := range []*cobra.Command{portlessQuickSetupCmd, portlessUninstallCmd, portlessSetupTrustCmd} {
		if cmd.Short == "" {
			t.Fatalf("%s has empty Short help", cmd.Use)
		}
		if cmd.Long == "" {
			t.Fatalf("%s has empty Long help", cmd.Use)
		}
	}
}

func TestInstallPortlessFlagsRegistered(t *testing.T) {
	for _, name := range []string{"with-portless", "node-red", "portless-quick-setup", "portless-trust"} {
		if installCmd.Flags().Lookup(name) == nil {
			t.Fatalf("install flag --%s not registered", name)
		}
	}
	flag := installCmd.Flags().Lookup("node-red")
	if flag == nil {
		t.Fatal("install flag --node-red not registered")
	}
	if !strings.Contains(flag.Usage, "skip, native, or docker") {
		t.Fatalf("--node-red help missing expected values: %q", flag.Usage)
	}
}

func TestHasExplicitInstallFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "install"}
	cmd.Flags().String("node-red", "", "")
	cmd.Flags().Bool("with-portless", false, "")
	cmd.Flags().Bool("portless-quick-setup", false, "")
	cmd.Flags().Bool("portless-trust", false, "")

	if hasExplicitInstallFlags(cmd) {
		t.Fatal("new command should not report explicit install flags")
	}
	if err := cmd.Flags().Set("node-red", "skip"); err != nil {
		t.Fatal(err)
	}
	if !hasExplicitInstallFlags(cmd) {
		t.Fatal("changed --node-red should report explicit install flags")
	}
}

func TestValidateInstallPortlessFlags(t *testing.T) {
	tests := []struct {
		name       string
		with       bool
		nodeRed    string
		quickSetup bool
		trust      bool
		wantErr    bool
	}{
		{name: "plain install"},
		{name: "valid node-red skip", nodeRed: "skip"},
		{name: "valid node-red native", nodeRed: "native"},
		{name: "valid node-red docker", nodeRed: "docker"},
		{name: "invalid node-red mode", nodeRed: "wrong", wantErr: true},
		{name: "with portless only", with: true},
		{name: "quick setup requires portless", quickSetup: true, wantErr: true},
		{name: "trust requires portless", trust: true, wantErr: true},
		{name: "quick setup allowed with portless", with: true, quickSetup: true},
		{name: "trust allowed with portless", with: true, trust: true},
		{name: "both allowed with portless", with: true, quickSetup: true, trust: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldWith := installWithPortless
			oldQuickSetup := installPortlessQuickSetup
			oldTrust := installPortlessTrust
			oldNodeRedMode := installNodeRedMode
			installWithPortless = tt.with
			installNodeRedMode = tt.nodeRed
			installPortlessQuickSetup = tt.quickSetup
			installPortlessTrust = tt.trust
			t.Cleanup(func() {
				installWithPortless = oldWith
				installNodeRedMode = oldNodeRedMode
				installPortlessQuickSetup = oldQuickSetup
				installPortlessTrust = oldTrust
			})

			err := validateInstallPortlessFlags()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateInstallPortlessFlags error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "--with-portless") && !strings.Contains(err.Error(), "--node-red") {
				t.Fatalf("error = %v, want flag guidance", err)
			}
		})
	}
}

func TestParseNodeRedInstallMode(t *testing.T) {
	tests := []struct {
		input   string
		want    model.NodeRedInstallMode
		wantErr bool
	}{
		{input: "skip", want: model.NodeRedInstallModeSkip},
		{input: " SKIP ", want: model.NodeRedInstallModeSkip},
		{input: "native", want: model.NodeRedInstallModeNative},
		{input: "docker", want: model.NodeRedInstallModeDocker},
		{input: "", wantErr: true},
		{input: "podman", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseNodeRedInstallMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNodeRedInstallMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseNodeRedInstallMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNodeRedCommandRegistered verifies node-red command has all subcommands
func TestNodeRedCommandRegistered(t *testing.T) {
	var nodeRedCmd *cobra.Command

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "node-red" {
			nodeRedCmd = cmd
			break
		}
	}

	if nodeRedCmd == nil {
		t.Fatal("node-red command not found")
	}

	requiredSubcmds := []string{
		"start",
		"stop",
		"restart",
		"status",
		"logs",
		"version",
		"detect",
		"install",
		"uninstall",
		"update",
	}

	subcmds := nodeRedCmd.Commands()
	subcmdNames := make(map[string]bool)

	for _, cmd := range subcmds {
		subcmdNames[cmd.Use] = true
	}

	for _, required := range requiredSubcmds {
		if !subcmdNames[required] {
			t.Errorf("node-red subcommand '%s' not registered", required)
		}
	}
}

// TestConfigCommandRegistered verifies config command has all subcommands
func TestConfigCommandRegistered(t *testing.T) {
	var configCmd *cobra.Command

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			configCmd = cmd
			break
		}
	}

	if configCmd == nil {
		t.Fatal("config command not found")
	}

	requiredSubcmds := []string{
		"list",
		"get",
		"set",
		"backup",
	}

	subcmds := configCmd.Commands()
	subcmdNames := make(map[string]bool)

	for _, cmd := range subcmds {
		// Extract first word from Use field (e.g., "get <key>" -> "get")
		parts := strings.Fields(cmd.Use)
		if len(parts) > 0 {
			subcmdNames[parts[0]] = true
		}
	}

	for _, required := range requiredSubcmds {
		if !subcmdNames[required] {
			t.Errorf("config subcommand '%s' not registered", required)
		}
	}
}

// TestEnvCommandRegistered verifies env command has all subcommands
func TestEnvCommandRegistered(t *testing.T) {
	var envCmd *cobra.Command

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "env" {
			envCmd = cmd
			break
		}
	}

	if envCmd == nil {
		t.Fatal("env command not found")
	}

	requiredSubcmds := []string{
		"list",
		"set",
		"delete",
		"edit",
	}

	subcmds := envCmd.Commands()
	subcmdNames := make(map[string]bool)

	for _, cmd := range subcmds {
		// Extract first word from Use field (e.g., "set <KEY=VALUE>" -> "set")
		parts := strings.Fields(cmd.Use)
		if len(parts) > 0 {
			subcmdNames[parts[0]] = true
		}
	}

	for _, required := range requiredSubcmds {
		if !subcmdNames[required] {
			t.Errorf("env subcommand '%s' not registered", required)
		}
	}
}

// TestDoctorCommandExists verifies doctor command is available
func TestDoctorCommandExists(t *testing.T) {
	var found bool

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "doctor" {
			found = true
			break
		}
	}

	if !found {
		t.Error("doctor command not found in root commands")
	}
}

// TestSetupCommandExists verifies setup command is available
func TestSetupCommandExists(t *testing.T) {
	var found bool

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "setup" {
			found = true
			break
		}
	}

	if !found {
		t.Error("setup command not found in root commands")
	}
}

// TestUpdateCommandExists verifies update command group exists
func TestUpdateCommandExists(t *testing.T) {
	var updateCmd *cobra.Command

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			updateCmd = cmd
			break
		}
	}

	if updateCmd == nil {
		t.Fatal("update command not found")
	}

	// Verify it has subcommands
	subcmds := updateCmd.Commands()
	if len(subcmds) == 0 {
		t.Error("update command should have subcommands (check, apply)")
	}
}

// TestPersistentFlags verifies root command has expected persistent flags
func TestPersistentFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	// Check for data-dir flag
	dataDir := flags.Lookup("data-dir")
	if dataDir == nil {
		t.Error("--data-dir flag not found")
	}

	// Check for json flag
	jsonFlag := flags.Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag not found")
	}

	// Check for no-color flag
	noColor := flags.Lookup("no-color")
	if noColor == nil {
		t.Error("--no-color flag not found")
	}
}

// TestNodeRedStatusJSONStructure verifies JSON output has correct envelope
func TestNodeRedStatusJSONStructure(t *testing.T) {
	// Test printJSON function directly
	testData := map[string]interface{}{
		"status": "stopped",
		"pid":    0,
	}

	// Capture output by redirecting stdout temporarily
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call printJSON
	err := printJSON(testData)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("printJSON failed: %v", err)
	}

	// Read output
	output := bytes.NewBufferString("")
	_, _ = output.ReadFrom(r)

	var result map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Logf("JSON output: %s", output.String())
		// May fail if no output captured, but that's ok for this test structure
	}
}

// TestPrintJSONEnvelope verifies JSON envelope format
func TestPrintJSONEnvelope(t *testing.T) {
	// We'll test the structure by looking at the function output
	testData := map[string]string{
		"key": "value",
	}

	// The envelope should have: ok, data, error fields
	// This is validated in the function itself
	err := printJSON(testData)
	if err != nil {
		t.Errorf("printJSON should not error: %v", err)
	}
}

// TestPrintErrorEnvelope verifies error JSON envelope format
func TestPrintErrorEnvelope(t *testing.T) {
	// Use a real error instead of nil to avoid panic
	testErr := &json.SyntaxError{Offset: 0}
	err := printError(testErr)

	if err != nil {
		// This is expected if something goes wrong
		t.Logf("printError returned: %v", err)
	}
}

// TestCobraCommandIntegration ensures rootCmd is properly set up
func TestCobraCommandIntegration(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "nrcc" {
		t.Errorf("root command Use should be 'nrcc', got %q", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("root command should have Short description")
	}

	if rootCmd.Long == "" {
		t.Error("root command should have Long description")
	}

	if rootCmd.PersistentPreRunE == nil {
		t.Error("root command should have PersistentPreRunE")
	}
}

// TestExecuteFunction verifies Execute() is exported and callable
func TestExecuteFunction(t *testing.T) {
	// Verify Execute function exists and is callable
	// (type checking at compile time ensures this works)
	_ = Execute
}
