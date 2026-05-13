package ui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// Init configures pterm defaults based on terminal capabilities and environment.
// Call this once at the top of main() before any other output.
func Init() {
	// pterm automatically detects TTY and respects NO_COLOR env var.
	// If stdout is not a TTY or NO_COLOR is set, pterm disables colors and styling.
}

// Banner renders a styled startup box with app name, version, port, and data directory.
func Banner(version, port, dataDir string) {
	content := fmt.Sprintf("Version: %s\nListening on: :%s\nData directory: %s",
		version, port, dataDir)
	pterm.DefaultBox.WithTitle("nrcc").Print(content)
}

// SectionHeader renders a visually distinct phase header line.
func SectionHeader(title string) {
	pterm.DefaultSection.Println(title)
}

// Info logs an informational message with green [INFO] prefix (or plain text in non-TTY).
func Info(msg string) {
	pterm.Info.Println(msg)
}

// Infof logs a formatted informational message.
func Infof(format string, args ...any) {
	pterm.Info.Printfln(format, args...)
}

// Warn logs a warning message with yellow [WARN] prefix.
func Warn(msg string) {
	pterm.Warning.Println(msg)
}

// Warnf logs a formatted warning message.
func Warnf(format string, args ...any) {
	pterm.Warning.Printfln(format, args...)
}

// Error logs an error message with red [ERROR] prefix.
func Error(msg string) {
	pterm.Error.Println(msg)
}

// Errorf logs a formatted error message.
func Errorf(format string, args ...any) {
	pterm.Error.Printfln(format, args...)
}

// Debug logs a debug message with muted [DEBUG] prefix.
func Debug(msg string) {
	pterm.Debug.Println(msg)
}

// DoctorRow is the display model for a single dependency row.
type DoctorRow struct {
	Name      string
	Installed bool
	Version   string
	Command   string
	Details   string
}

// DoctorTable renders the host status as a pterm table with dependency rows.
// Each row shows: dependency name, ✓/✗ status marker, version, and command path.
func DoctorTable(rows []DoctorRow) {
	// Build table data
	tableData := [][]string{
		{"Dependency", "Status", "Version", "Command", "Details"},
	}

	for _, row := range rows {
		status := "✓"
		if !row.Installed {
			status = "✗"
		}
		tableData = append(tableData, []string{
			row.Name,
			status,
			row.Version,
			row.Command,
			row.Details,
		})
	}

	// Render table with pterm
	table := pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData)
	table.Render()
}

// StartSpinner starts and returns a spinner with the given message.
// Caller is responsible for calling .Success(msg) or .Fail(msg) to stop it.
func StartSpinner(msg string) *pterm.SpinnerPrinter {
	spinner, _ := pterm.DefaultSpinner.Start(msg)
	return spinner
}

// HTTPLog emits one structured HTTP request log line with status-code-driven color.
// Status codes: 2xx=green, 3xx=cyan, 4xx=yellow, 5xx=red.
func HTTPLog(method, path string, statusCode int, duration time.Duration) {
	statusStr := fmt.Sprintf("%d", statusCode)
	durationMs := duration.Milliseconds()

	// Determine color prefix based on status code
	var coloredStatus string
	switch {
	case statusCode >= 200 && statusCode < 300:
		// 2xx - green
		coloredStatus = pterm.Green(statusStr)
	case statusCode >= 300 && statusCode < 400:
		// 3xx - cyan
		coloredStatus = pterm.Cyan(statusStr)
	case statusCode >= 400 && statusCode < 500:
		// 4xx - yellow
		coloredStatus = pterm.Yellow(statusStr)
	case statusCode >= 500:
		// 5xx - red
		coloredStatus = pterm.Red(statusStr)
	default:
		coloredStatus = pterm.White(statusStr)
	}

	// Format: METHOD PATH STATUS DURATION
	// Pad method to 6 chars for alignment
	methodPad := fmt.Sprintf("%-6s", method)
	logLine := fmt.Sprintf("%s %s %s %dms", methodPad, path, coloredStatus, durationMs)

	// Print via pterm Info to maintain consistency
	pterm.Info.Println(logLine)
}
