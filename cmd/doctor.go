package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/security"
	"nrcc/internal/service"
)

// ColorCode struct for terminal colors
type ColorCode struct {
	Reset  string
	Green  string
	Yellow string
	Red    string
	Bold   string
}

// getColors returns color codes if terminal supports it, otherwise empty strings
func getColors(w io.Writer) ColorCode {
	// Simple check: if stdout is terminal-like (not ideal but works for basic checks)
	isTTY := false
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err == nil {
			// Check if it's a character device (terminal)
			mode := stat.Mode()
			isTTY = (mode & os.ModeCharDevice) != 0
		}
	}

	if !isTTY || os.Getenv("NO_COLOR") != "" {
		return ColorCode{}
	}

	return ColorCode{
		Reset:  "\033[0m",
		Green:  "\033[32m",
		Yellow: "\033[33m",
		Red:    "\033[31m",
		Bold:   "\033[1m",
	}
}

// runDoctor handles the "nrcc doctor" command
func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonFlag := fs.Bool("json", false, "output as JSON")
	exportFlag := fs.Bool("export", false, "export support bundle")
	fs.Parse(args)

	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}

	// Initialize doctor service
	doctorService := service.NewDoctorService(dataDir)

	// Run the doctor checks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := doctorService.Run(ctx)

	// Handle --export flag (generate support bundle)
	if *exportFlag {
		// Initialize required services for support bundle
		authService, err := service.NewAuthService(dataDir)
		if err != nil {
			return err
		}
		defer authService.Close()

		logService, err := service.NewLogService(dataDir, authService.GetDB())
		if err != nil {
			return err
		}
		defer logService.Close()

		sanitizer := security.NewSanitizer()
		supportBundleService := service.NewSupportBundleService(dataDir, logService, doctorService, sanitizer)

		bundlePath, err := supportBundleService.Export(ctx)
		if err != nil {
			return fmt.Errorf("failed to export support bundle: %w", err)
		}

		fmt.Printf("Support bundle created: %s\n", bundlePath)
		return nil
	}

	// Handle --json flag
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}

	// Default: human-readable output with colors
	printDoctorReport(report, os.Stdout)
	return nil
}

// printDoctorReport prints the doctor report in human-readable format
func printDoctorReport(report model.DoctorReport, w io.Writer) {
	color := getColors(w)

	// Header
	fmt.Fprintf(w, "\n%sNode-RED Control Center — Doctor Report%s\n", color.Bold, color.Reset)
	fmt.Fprintf(w, "========================================\n")

	// Counters for summary
	passCount := 0
	warnCount := 0
	failCount := 0

	// Print each check
	for _, check := range report.Checks {
		var icon, checkColor string

		switch check.Status {
		case model.CheckStatusPass:
			icon = "✅"
			checkColor = color.Green
			passCount++
		case model.CheckStatusWarn:
			icon = "⚠️"
			checkColor = color.Yellow
			warnCount++
		case model.CheckStatusFail:
			icon = "❌"
			checkColor = color.Red
			failCount++
		default:
			icon = "?"
			checkColor = color.Yellow
		}

		fmt.Fprintf(w, "%s %s%-20s%s %s\n", icon, checkColor, check.Label, color.Reset, check.Message)
	}

	// Footer
	fmt.Fprintf(w, "----------------------------------------\n")

	// Overall status
	overallStr := fmt.Sprintf("Overall: %s", report.OverallStatus)
	summary := fmt.Sprintf("(%d pass, %d warning, %d fail)", passCount, warnCount, failCount)

	var overallColor string
	switch report.OverallStatus {
	case model.OverallHealthy:
		overallColor = color.Green
	case model.OverallDegraded:
		overallColor = color.Yellow
	case model.OverallCritical:
		overallColor = color.Red
	default:
		overallColor = color.Yellow
	}

	fmt.Fprintf(w, "%s%s%s %s\n", overallColor, overallStr, color.Reset, summary)
	fmt.Fprintf(w, "\n")
}
